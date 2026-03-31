// Package provider реализует провайдеры данных для LogT.
package provider

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	Watch(paths []string) error
}

// MultiProvider объединяет несколько провайдеров в один.
type MultiProvider struct {
	providers []Provider
	logChan   chan domain.LogLine
	buffer    *domain.RingBuffer
	mu        sync.RWMutex
}

// NewMultiProvider создаёт новый MultiProvider.
func NewMultiProvider() *MultiProvider {
	return &MultiProvider{
		providers: make([]Provider, 0),
		logChan:   make(chan domain.LogLine, 1000),
		buffer:    domain.NewRingBuffer(5000),
	}
}

// AddProvider добавляет провайдер в MultiProvider.
func (mp *MultiProvider) AddProvider(p Provider) {
	mp.mu.Lock()
	mp.providers = append(mp.providers, p)
	mp.mu.Unlock()

	go mp.forwardLogs(p)
}

// forwardLogs пересылает логи из провайдера в основной канал.
func (mp *MultiProvider) forwardLogs(p Provider) {
	for logLine := range p.LogChan() {
		select {
		case mp.logChan <- logLine:
			mp.buffer.Add(logLine)
		default:
			mp.buffer.Add(logLine)
		}
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
		for k, v := range p.EnabledSources() {
			enabled[k] = v
		}
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
	defer mp.mu.Unlock()

	for _, p := range mp.providers {
		p.Close()
	}
	close(mp.logChan)
	return nil
}

// Watch запускает watching на всех провайдерах.
func (mp *MultiProvider) Watch(paths []string) error {
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
	closed         bool
}

// NewFileProvider создаёт новый FileProvider.
func NewFileProvider() *FileProvider {
	return &FileProvider{
		parser:         domain.NewMultiParser(),
		logChan:        make(chan domain.LogLine, 1000),
		sources:        make(map[string]*os.File),
		includeSources: make(map[string]bool),
		offsets:        make(map[string]int64),
	}
}

// Watch начинает слежение за файлами по указанным путям.
func (fp *FileProvider) Watch(paths []string) error {
	for _, pathPattern := range paths {
		matches, err := filepath.Glob(pathPattern)
		if err != nil {
			return fmt.Errorf("invalid glob pattern %s: %w", pathPattern, err)
		}
		for _, path := range matches {
			if err := fp.watchFile(path); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to watch %s: %v\n", path, err)
			}
		}
	}
	return nil
}

// watchFile открывает файл и начинает за ним следить.
func (fp *FileProvider) watchFile(path string) error {
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

	go fp.watchLoop(path, file, true)

	return nil
}

// watchLoop основной цикл слежения за файлом.
func (fp *FileProvider) watchLoop(path string, file *os.File, initialRead bool) {
	source := domain.Source{
		Name: filepath.Base(path),
		Path: path,
	}

	if initialRead {
		fp.readExistingContent(file, source)
	}

	reader := bufio.NewReader(file)
	currentOffset := fp.getOffset(path)

	for {
		newSize, err := fp.getFileSize(file)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if newSize < currentOffset {
			currentOffset = 0
			reader = fp.resetReader(file)
		}

		if newSize > currentOffset {
			currentOffset = fp.readNewLines(file, reader, currentOffset, source, path)
		}

		time.Sleep(100 * time.Millisecond)
	}
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
	file.Seek(0, io.SeekStart)
	return bufio.NewReader(file)
}

// readNewLines читает новые строки и возвращает новое смещение.
func (fp *FileProvider) readNewLines(file *os.File, reader *bufio.Reader, currentOffset int64, source domain.Source, path string) int64 {
	file.Seek(currentOffset, io.SeekStart)
	reader = bufio.NewReader(file)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		line = strings.TrimSuffix(line, "\n")
		line = strings.TrimSuffix(line, "\r")

		if line != "" {
			logLine := fp.parser.Parse(line, source)
			if logLine != nil {
				select {
				case fp.logChan <- *logLine:
				case <-time.After(10 * time.Millisecond):
				}
			}
		}
	}

	newOffset, _ := file.Seek(0, io.SeekCurrent)
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
func (fp *FileProvider) readExistingContent(file *os.File, source domain.Source) {
	domain.ReadExistingContent(file, source, fp.parser, fp.logChan)
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

	enabled := make(map[string]bool)
	for k, v := range fp.includeSources {
		enabled[k] = v
	}
	return enabled
}

// Close закрывает FileProvider.
func (fp *FileProvider) Close() error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if fp.closed {
		return nil
	}
	fp.closed = true

	for path, file := range fp.sources {
		file.Close()
		delete(fp.sources, path)
	}
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
}

// NewStdinProvider создаёт новый StdinProvider.
func NewStdinProvider() *StdinProvider {
	return &StdinProvider{
		parser:  domain.NewMultiParser(),
		logChan: make(chan domain.LogLine, 1000),
		reader:  bufio.NewReader(os.Stdin),
	}
}

// Start запускает чтение из stdin.
func (sp *StdinProvider) Start() error {
	sp.mu.Lock()
	if sp.closed {
		sp.mu.Unlock()
		return nil
	}
	sp.mu.Unlock()

	source := domain.Source{
		Name: "stdin",
		Path: "stdin",
	}

	go sp.readLines(source)

	return nil
}

// readLines читает строки из stdin.
func (sp *StdinProvider) readLines(source domain.Source) {
	scanner := bufio.NewScanner(sp.reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		logLine := sp.parser.Parse(line, source)
		if logLine != nil {
			select {
			case sp.logChan <- *logLine:
			case <-time.After(10 * time.Millisecond):
			}
		}
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

// Close закрывает StdinProvider.
func (sp *StdinProvider) Close() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if !sp.closed {
		sp.closed = true
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
func (sp *StdinProvider) Watch(paths []string) error {
	return sp.Start()
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
			matches, _ := filepath.Glob(p)
			result = append(result, matches...)
		}
	}
	return result
}
