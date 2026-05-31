// Package provider реализует провайдеры данных для LogT.
package provider

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/turkprogrammer/logt/internal/domain"
)

// Provider определяет интерфейс для источников логов.
type Provider interface {
	LogChan() <-chan domain.LogLine
	Close() error
	Sources() []domain.Source
	ToggleSource(path string)
	EnabledSources() map[string]bool
	IsSourceEnabled(path string) bool
	Watch(ctx context.Context, paths []string) error
	Buffer() *domain.RingBuffer
}

// MultiProvider объединяет несколько провайдеров в один.
type MultiProvider struct {
	providers []Provider
	logChan   chan domain.LogLine
	buffer    *domain.RingBuffer
	mu        sync.RWMutex
	wg        sync.WaitGroup
}

// NewMultiProvider создаёт новый MultiProvider.
func NewMultiProvider() *MultiProvider {
	return &MultiProvider{
		logChan: make(chan domain.LogLine, 1000),
		buffer:  domain.NewRingBuffer(5000),
	}
}

// AddProvider добавляет провайдер в MultiProvider.
func (mp *MultiProvider) AddProvider(p Provider) {
	mp.mu.Lock()
	mp.providers = append(mp.providers, p)
	mp.mu.Unlock()

	mp.wg.Add(1)
	go mp.forwardLogs(p)
}

// forwardLogs пересылает логи из провайдера в основной канал.
func (mp *MultiProvider) forwardLogs(p Provider) {
	defer mp.wg.Done()
	for logLine := range p.LogChan() {
		mp.logChan <- logLine
		mp.buffer.Add(logLine)
	}
}

// LogChan возвращает объединённый канал для получения логов.
func (mp *MultiProvider) LogChan() <-chan domain.LogLine {
	return mp.logChan
}

// Buffer возвращает ring буфер логов.
func (mp *MultiProvider) Buffer() *domain.RingBuffer {
	return mp.buffer
}

// Sources возвращает список всех источников из всех провайдеров.
func (mp *MultiProvider) Sources() []domain.Source {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	seen := make(map[string]bool)
	var sources []domain.Source

	for _, p := range mp.providers {
		for _, s := range p.Sources() {
			if !seen[s.Path] {
				seen[s.Path] = true
				sources = append(sources, s)
			}
		}
	}
	return sources
}

// ToggleSource переключает источник во всех провайдерах.
func (mp *MultiProvider) ToggleSource(path string) {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	for _, p := range mp.providers {
		p.ToggleSource(path)
	}
}

// EnabledSources возвращает карту включённых источников.
func (mp *MultiProvider) EnabledSources() map[string]bool {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	enabled := make(map[string]bool)
	for _, p := range mp.providers {
		maps.Copy(enabled, p.EnabledSources())
	}
	return enabled
}

// IsSourceEnabled проверяет, включён ли источник.
func (mp *MultiProvider) IsSourceEnabled(path string) bool {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	for _, p := range mp.providers {
		if p.IsSourceEnabled(path) {
			return true
		}
	}
	return false
}

// Close закрывает все провайдеры.
func (mp *MultiProvider) Close() error {
	mp.mu.Lock()
	for _, p := range mp.providers {
		p.Close()
	}
	mp.mu.Unlock()

	mp.wg.Wait()
	close(mp.logChan)
	return nil
}

// Watch запускает watching на всех провайдерах.
func (mp *MultiProvider) Watch(ctx context.Context, paths []string) error {
	// MultiProvider не watchит пути напрямую,
	// это делают добавленные в него провайдеры
	return nil
}

// FileProvider читает логи из файлов с поддержкой tail.
type FileProvider struct {
	parser         *domain.MultiParser
	logChan        chan domain.LogLine
	sources        map[string]*os.File
	mu             sync.RWMutex
	includeSources map[string]bool
	offsets        map[string]int64
	closed         atomic.Bool
	done           chan struct{}
	wg             sync.WaitGroup
}

// NewFileProvider создаёт новый FileProvider.
func NewFileProvider() *FileProvider {
	return &FileProvider{
		parser:         domain.NewMultiParser(),
		logChan:        make(chan domain.LogLine, 1000),
		sources:        make(map[string]*os.File),
		includeSources: make(map[string]bool),
		offsets:        make(map[string]int64),
		done:           make(chan struct{}),
	}
}

// Buffer возвращает nil для FileProvider (буфер управляется MultiProvider).
func (fp *FileProvider) Buffer() *domain.RingBuffer {
	return nil
}

// Watch начинает слежение за файлами по указанным путям.
func (fp *FileProvider) Watch(ctx context.Context, paths []string) error {
	for _, pathPattern := range paths {
		matches, err := filepath.Glob(pathPattern)
		if err != nil {
			return fmt.Errorf("invalid glob pattern %s: %w", pathPattern, err)
		}
		for _, path := range matches {
			if err := fp.watchFile(ctx, path); err != nil {
				slog.Warn("failed to watch file", "path", path, "error", err)
			}
		}
	}
	return nil
}

// watchFile открывает файл и начинает за ним следить.
func (fp *FileProvider) watchFile(ctx context.Context, path string) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if _, exists := fp.sources[path]; exists {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}

	fp.sources[path] = file
	fp.includeSources[path] = true
	fp.offsets[path] = stat.Size()

	fp.wg.Add(1)
	go fp.watchLoop(ctx, path, file, true)

	return nil
}

// watchLoop основной цикл слежения за файлом.
func (fp *FileProvider) watchLoop(ctx context.Context, path string, file *os.File, initialRead bool) {
	defer fp.wg.Done()

	source := domain.Source{
		Name: filepath.Base(path),
		Path: path,
	}

	if initialRead {
		fp.readExistingContent(ctx, file, source)
	}

	reader := bufio.NewReader(file)
	currentOffset := fp.getOffset(path)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-fp.done:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		newSize, err := fp.getFileSize(file)
		if err != nil {
			if fp.isClosed() {
				return
			}
			continue
		}

		if newSize < currentOffset {
			currentOffset = 0
			reader = fp.resetReader(file)
		}

		if newSize > currentOffset {
			currentOffset = fp.readNewLines(ctx, file, reader, currentOffset, source, path)
		}
	}
}

// isClosed проверяет, закрыт ли провайдер.
func (fp *FileProvider) isClosed() bool {
	return fp.closed.Load()
}

// getOffset возвращает смещение для файла.
func (fp *FileProvider) getOffset(path string) int64 {
	fp.mu.RLock()
	defer fp.mu.RUnlock()
	return fp.offsets[path]
}

// getFileSize получает размер файла.
func (fp *FileProvider) getFileSize(file *os.File) (int64, error) {
	fileInfo, err := file.Stat()
	if err != nil {
		return 0, err
	}
	return fileInfo.Size(), nil
}

// resetReader сбрасывает reader после ротации файла.
func (fp *FileProvider) resetReader(file *os.File) *bufio.Reader {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return bufio.NewReader(file)
	}
	return bufio.NewReader(file)
}

// sendWithTimeout отправляет лог в канал с таймаутом 10ms.
func (fp *FileProvider) sendWithTimeout(ctx context.Context, logLine domain.LogLine) {
	t := time.NewTimer(10 * time.Millisecond)
	defer t.Stop()
	select {
	case fp.logChan <- logLine:
	case <-t.C:
	case <-ctx.Done():
	}
}

// readNewLines читает новые строки и возвращает новое смещение.
func (fp *FileProvider) readNewLines(ctx context.Context, file *os.File, reader *bufio.Reader, currentOffset int64, source domain.Source, path string) int64 {
	if _, err := file.Seek(currentOffset, io.SeekStart); err != nil {
		return currentOffset
	}
	reader = bufio.NewReader(file)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			break
		}

		line = strings.TrimSuffix(line, "\n")
		line = strings.TrimSuffix(line, "\r")

		if line != "" {
			logLine := fp.parser.Parse(line, source)
			if logLine != nil {
				fp.sendWithTimeout(ctx, *logLine)
			}
		}

		if err := ctx.Err(); err != nil {
			break
		}
	}

	newOffset, seekErr := file.Seek(0, io.SeekCurrent)
	if seekErr != nil {
		return currentOffset
	}
	fp.updateOffset(path, newOffset)
	return newOffset
}

// updateOffset обновляет смещение для файла.
func (fp *FileProvider) updateOffset(path string, offset int64) {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	fp.offsets[path] = offset
}

// readExistingContent читает весь существующий контент файла.
func (fp *FileProvider) readExistingContent(ctx context.Context, file *os.File, source domain.Source) {
	domain.ReadExistingContent(ctx, file, source, fp.parser, fp.logChan)
}

// LogChan возвращает канал для получения логов.
func (fp *FileProvider) LogChan() <-chan domain.LogLine {
	return fp.logChan
}

// Sources возвращает список открытых файлов.
func (fp *FileProvider) Sources() []domain.Source {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	sources := make([]domain.Source, 0, len(fp.sources))
	for path := range fp.sources {
		sources = append(sources, domain.Source{
			Name: filepath.Base(path),
			Path: path,
		})
	}
	return sources
}

// ToggleSource переключает отображение источника.
func (fp *FileProvider) ToggleSource(path string) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if fp.includeSources[path] {
		fp.includeSources[path] = false
	} else {
		fp.includeSources[path] = true
	}
}

// IsSourceEnabled проверяет, включён ли источник.
func (fp *FileProvider) IsSourceEnabled(path string) bool {
	fp.mu.RLock()
	defer fp.mu.RUnlock()
	return fp.includeSources[path]
}

// EnabledSources возвращает карту включённых источников.
func (fp *FileProvider) EnabledSources() map[string]bool {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	return maps.Clone(fp.includeSources)
}

// Close закрывает FileProvider.
func (fp *FileProvider) Close() error {
	if fp.closed.Load() {
		return nil
	}
	fp.closed.Store(true)

	close(fp.done)

	fp.mu.Lock()
	for path, file := range fp.sources {
		file.Close()
		delete(fp.sources, path)
	}
	fp.mu.Unlock()

	fp.wg.Wait()
	close(fp.logChan)
	return nil
}

// StdinProvider читает логи из stdin.
type StdinProvider struct {
	parser  *domain.MultiParser
	logChan chan domain.LogLine
	reader  *bufio.Reader
	mu      sync.Mutex
	closed  bool
	done    chan struct{}
}

// NewStdinProvider создаёт новый StdinProvider.
func NewStdinProvider() *StdinProvider {
	return &StdinProvider{
		parser:  domain.NewMultiParser(),
		logChan: make(chan domain.LogLine, 1000),
		reader:  bufio.NewReader(os.Stdin),
		done:    make(chan struct{}),
	}
}

// Start запускает чтение из stdin.
func (sp *StdinProvider) Start(ctx context.Context) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if sp.closed {
		return nil
	}

	source := domain.Source{
		Name: "stdin",
		Path: "stdin",
	}

	go sp.readLines(ctx, source)

	return nil
}

// readLines читает строки из stdin.
func (sp *StdinProvider) readLines(ctx context.Context, source domain.Source) {
	scanner := bufio.NewScanner(sp.reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		select {
		case <-sp.done:
			return
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		logLine := sp.parser.Parse(line, source)
		if logLine != nil {
			t := time.NewTimer(10 * time.Millisecond)
			select {
			case sp.logChan <- *logLine:
				if !t.Stop() {
					<-t.C
				}
			case <-t.C:
			case <-sp.done:
				if !t.Stop() {
					<-t.C
				}
				return
			case <-ctx.Done():
				if !t.Stop() {
					<-t.C
				}
				return
			}
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Error("stdin read error", "error", err)
	}

	sp.mu.Lock()
	if !sp.closed {
		sp.closed = true
		close(sp.logChan)
	}
	sp.mu.Unlock()
}

// LogChan возвращает канал для получения логов.
func (sp *StdinProvider) LogChan() <-chan domain.LogLine {
	return sp.logChan
}

// Buffer возвращает nil для StdinProvider (буфер управляется MultiProvider).
func (sp *StdinProvider) Buffer() *domain.RingBuffer {
	return nil
}

// Close закрывает StdinProvider.
func (sp *StdinProvider) Close() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if !sp.closed {
		sp.closed = true
		close(sp.done)
		close(sp.logChan)
	}
	return nil
}

// Sources возвращает единственный источник "stdin".
func (sp *StdinProvider) Sources() []domain.Source {
	return []domain.Source{{Name: "stdin", Path: "stdin"}}
}

// EnabledSources возвращает карту с включённым stdin.
func (sp *StdinProvider) EnabledSources() map[string]bool {
	return map[string]bool{"stdin": true}
}

// ToggleSource пустая реализация.
func (sp *StdinProvider) ToggleSource(path string) {
}

// IsSourceEnabled всегда возвращает true для stdin.
func (sp *StdinProvider) IsSourceEnabled(path string) bool {
	return path == "stdin"
}

// Watch для StdinProvider - заглушка (stdin не требует watching).
func (sp *StdinProvider) Watch(ctx context.Context, paths []string) error {
	return sp.Start(ctx)
}

// IsStdinPiped проверяет, подключён ли stdin к pipe.
func IsStdinPiped() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) == 0
}

// ExpandPaths раскрывает glob паттерны в список путей.
func ExpandPaths(paths []string) []string {
	var result []string
	for _, p := range paths {
		if len(p) > 0 {
			matches, err := filepath.Glob(p)
			if err != nil {
				slog.Warn("invalid glob pattern", "pattern", p, "error", err)
				continue
			}
			result = append(result, matches...)
		}
	}
	return result
}
