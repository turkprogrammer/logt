//go:build linux || darwin
// +build linux darwin

package provider

import "runtime"

// IsWatcherSupported возвращает true для Linux/macOS.
func IsWatcherSupported() bool {
	return true
}

// IsWatcherPreferred проверяет, предпочтительно ли использовать watcher.
// Для Linux/macOS возвращаем true, так как inotify/FSEvents эффективнее polling.
func IsWatcherPreferred() bool {
	return true
}

// GetOSName возвращает имя текущей ОС.
func GetOSName() string {
	return runtime.GOOS
}

// GetWatcherType возвращает тип вотчера для текущей ОС.
func GetWatcherType() string {
	switch runtime.GOOS {
	case "linux":
		return "inotify"
	case "darwin":
		return "FSEvents"
	default:
		return "unknown"
	}
}
