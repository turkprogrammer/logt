//go:build linux || darwin
// +build linux darwin

package provider

// IsWatcherSupported возвращает true для Linux/macOS.
func IsWatcherSupported() bool {
	return true
}

// IsWatcherPreferred проверяет, предпочтительно ли использовать watcher.
// Для Linux/macOS возвращаем true, так как inotify/FSEvents эффективнее polling.
func IsWatcherPreferred() bool {
	return true
}
