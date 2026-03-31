// Package ui предоставляет поддержку цветовых режимов для TUI.
package ui

import (
	"fmt"
	"strings"
)

// ColorMode определяет режим использования цветов.
type ColorMode int

// Режимы цветов.
const (
	ColorAuto   ColorMode = iota // Автоматически (только TUI)
	ColorAlways                  // Всегда с цветами
	ColorNever                   // Без цветов
)

// String возвращает строковое представление ColorMode.
func (m ColorMode) String() string {
	switch m {
	case ColorAuto:
		return "auto"
	case ColorAlways:
		return "always"
	case ColorNever:
		return "never"
	default:
		return "unknown"
	}
}

// ParseColorMode парсит строку в ColorMode.
// Поддерживает: "auto", "always", "never" (case-insensitive).
// Пустая строка эквивалентна "auto".
func ParseColorMode(s string) (ColorMode, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ColorAuto, nil
	}

	switch s {
	case "auto":
		return ColorAuto, nil
	case "always":
		return ColorAlways, nil
	case "never":
		return ColorNever, nil
	default:
		return ColorAuto, fmt.Errorf("invalid color mode: %s (supported: auto, always, never)", s)
	}
}

// ShouldUseColor определяет, нужно ли использовать цвета.
// Учитывает режим и является ли вывод терминалом.
func ShouldUseColor(mode ColorMode, isTerminal bool) bool {
	switch mode {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	case ColorAuto:
		return isTerminal
	default:
		return isTerminal
	}
}
