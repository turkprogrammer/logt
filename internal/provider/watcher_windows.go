//go:build windows
// +build windows

package provider

// NewWatcherProvider для Windows возвращаем FileProvider с polling.
// Используем polling вместо ReadDirectoryChangesW из-за его сложности
// и проблем с надёжностью на Windows.
func NewWatcherProvider() *FileProvider {
	return NewFileProvider()
}

// IsWatcherSupported возвращает false для Windows.
func IsWatcherSupported() bool {
	return false
}

// IsWatcherPreferred проверяет, предпочтительно ли использовать watcher.
// Для Windows всегда возвращаем false.
func IsWatcherPreferred() bool {
	return false
}

// GetWatcherType возвращает тип вотчера для Windows.
func GetWatcherType() string {
	return "polling"
}
