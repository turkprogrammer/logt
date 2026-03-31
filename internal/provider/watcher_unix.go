//go:build linux || darwin
// +build linux darwin

package provider

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/radovskyb/watcher"
	"github.com/turkprogrammer/logt/internal/domain"
)

// WatcherProvider использует нативные OS API (inotify/FSEvents) для слежения за файлами.
type WatcherProvider struct {
	parser         *domain.MultiParser
	logChan        chan domain.LogLine
	sources        map[string]bool
	mu             sync.RWMutex
	includeSources map[string]bool
	offsets        map[string]int64
	closed         bool
	watcher        *watcher.Watcher
}

// NewWatcherProvider создаёт новый WatcherProvider с использованием inotify (Linux) или FSEvents (macOS).
func NewWatcherProvider() *WatcherProvider {
	w := watcher.New()
	w.SetMaxEvents(1000)

	return &WatcherProvider{
		parser:         domain.NewMultiParser(),
		logChan:        make(chan domain.LogLine, 1000),
		sources:        make(map[string]bool),
		includeSources: make(map[string]bool),
		offsets:        make(map[string]int64),
		watcher:        w,
	}
}

// Watch начинает слежение за файлами по указанным путям.
func (wp *WatcherProvider) Watch(paths []string) error {
	for _, pathPattern := range paths {
		matches, err := filepath.Glob(pathPattern)
		if err != nil {
			return fmt.Errorf("invalid glob pattern %s: %w", pathPattern, err)
		}

		for _, path := range matches {
			absPath, err := filepath.Abs(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to resolve absolute path for %s: %v\n", path, err)
				absPath = path
			}

			if err := wp.addWatch(absPath); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to watch %s: %v\n", path, err)
			}
		}
	}

	go wp.watchLoop()

	return nil
}

// addWatch добавляет файл или директорию в watcher.
func (wp *WatcherProvider) addWatch(path string) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.sources[path] {
		return nil
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	if fileInfo.IsDir() {
		if err := wp.watcher.AddRecursive(path); err != nil {
			return err
		}
	} else {
		if err := wp.watcher.Add(path); err != nil {
			return err
		}
	}

	wp.sources[path] = true
	wp.includeSources[path] = true

	return nil
}

// watchLoop обрабатывает события от watcher.
func (wp *WatcherProvider) watchLoop() {
	// При первом запуске читаем существующие файлы
	wp.readExistingFiles()

	for {
		select {
		case event, ok := <-wp.watcher.Event:
			if !ok {
				return
			}

			if event.IsDir() {
				continue
			}

			path := event.Path()

			switch {
			case event.Op&watcher.Create == watcher.Create:
				wp.handleNewFile(path)
			case event.Op&watcher.Write == watcher.Write:
				wp.handleFileWrite(path)
			case event.Op&watcher.Remove == watcher.Remove:
				wp.handleFileRemove(path)
			case event.Op&watcher.Rename == watcher.Rename:
				wp.handleFileRename(path)
			}

		case err, ok := <-wp.watcher.Error:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)

		case <-wp.watcher.Closed:
			return
		}
	}
}

// readExistingFiles читает содержимое всех файлов при старте.
func (wp *WatcherProvider) readExistingFiles() {
	wp.mu.RLock()
	paths := make([]string, 0, len(wp.sources))
	for path := range wp.sources {
		paths = append(paths, path)
	}
	wp.mu.RUnlock()

	for _, path := range paths {
		wp.watchFile(path)
	}
}

// watchFile читает существующее содержимое файла при первом запуске.
func (wp *WatcherProvider) watchFile(path string) {
	wp.mu.Lock()
	if _, exists := wp.offsets[path]; exists {
		wp.mu.Unlock()
		return
	}
	wp.mu.Unlock()

	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return
	}

	source := domain.Source{
		Name: filepath.Base(path),
		Path: path,
	}

	domain.ReadExistingContent(file, source, wp.parser, wp.logChan)

	wp.mu.Lock()
	wp.offsets[path] = stat.Size()
	wp.mu.Unlock()
}

// handleNewFile обрабатывает создание нового файла.
func (wp *WatcherProvider) handleNewFile(path string) {
	wp.mu.Lock()
	if wp.sources[path] {
		wp.mu.Unlock()
		return
	}
	wp.sources[path] = true
	wp.includeSources[path] = true
	wp.mu.Unlock()

	wp.watchFile(path)
}

// handleFileWrite обрабатывает запись в файл.
func (wp *WatcherProvider) handleFileWrite(path string) {
	if !wp.isSourceEnabled(path) {
		return
	}

	currentOffset := wp.getOffset(path)
	newSize, file, err := wp.openFileAndGetStat(path)
	if err != nil {
		return
	}
	defer file.Close()

	if newSize < currentOffset {
		currentOffset = 0
	}

	if newSize > currentOffset {
		wp.readAndSendLines(file, currentOffset, path)
		wp.updateOffset(path, newSize)
	}
}

// isSourceEnabled проверяет, включён ли источник.
func (wp *WatcherProvider) isSourceEnabled(path string) bool {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.includeSources[path]
}

// getOffset возвращает текущее смещение для файла.
func (wp *WatcherProvider) getOffset(path string) int64 {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.offsets[path]
}

// openFileAndGetStat открывает файл и получает его размер.
func (wp *WatcherProvider) openFileAndGetStat(path string) (int64, *os.File, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, nil, err
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return 0, nil, err
	}

	return stat.Size(), file, nil
}

// readAndSendLines читает новые строки и отправляет в канал.
func (wp *WatcherProvider) readAndSendLines(file *os.File, offset int64, path string) {
	file.Seek(offset, 0)
	reader := bufio.NewReader(file)

	source := domain.Source{
		Name: filepath.Base(path),
		Path: path,
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.TrimSuffix(line, "\n")
		line = strings.TrimSuffix(line, "\r")

		if line != "" {
			logLine := wp.parser.Parse(line, source)
			if logLine != nil {
				select {
				case wp.logChan <- *logLine:
				case <-time.After(10 * time.Millisecond):
				}
			}
		}
	}
}

// updateOffset обновляет смещение для файла.
func (wp *WatcherProvider) updateOffset(path string, newSize int64) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	wp.offsets[path] = newSize
}

// handleFileRemove обрабатывает удаление файла.
func (wp *WatcherProvider) handleFileRemove(path string) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	delete(wp.sources, path)
	delete(wp.includeSources, path)
	delete(wp.offsets, path)
}

// handleFileRename обрабатывает переименование файла.
func (wp *WatcherProvider) handleFileRename(oldPath string) {
	wp.handleFileRemove(oldPath)
}

// LogChan возвращает канал для получения логов.
func (wp *WatcherProvider) LogChan() <-chan domain.LogLine {
	return wp.logChan
}

// Sources возвращает список источников.
func (wp *WatcherProvider) Sources() []domain.Source {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	sources := make([]domain.Source, 0, len(wp.sources))
	for path := range wp.sources {
		sources = append(sources, domain.Source{
			Name: filepath.Base(path),
			Path: path,
		})
	}
	return sources
}

// ToggleSource переключает отображение источника.
func (wp *WatcherProvider) ToggleSource(path string) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.includeSources[path] {
		wp.includeSources[path] = false
	} else {
		wp.includeSources[path] = true
	}
}

// IsSourceEnabled проверяет, включён ли источник.
func (wp *WatcherProvider) IsSourceEnabled(path string) bool {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.includeSources[path]
}

// EnabledSources возвращает карту включённых источников.
func (wp *WatcherProvider) EnabledSources() map[string]bool {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	enabled := make(map[string]bool)
	for k, v := range wp.includeSources {
		enabled[k] = v
	}
	return enabled
}

// Close закрывает WatcherProvider.
func (wp *WatcherProvider) Close() error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.closed {
		return nil
	}
	wp.closed = true

	if err := wp.watcher.Close(); err != nil {
		return err
	}

	close(wp.logChan)
	return nil
}
