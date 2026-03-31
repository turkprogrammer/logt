// Package domain реализует парсинг времени для фильтрации логов.
package domain

import (
	"fmt"
	"time"
)

// ParseSince парсит относительное или абсолютное время.
// Поддерживаемые форматы:
//   - Относительное: 1h, 30m, 24h (1 день)
//   - Абсолютное: 2024-01-15 10:00, 2024-01-15
//   - ISO8601: 2024-01-15T10:00:00, 2024-01-15T10:00:00Z
//
// Для относительного времени вычитает длительность из текущего времени.
func ParseSince(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time string")
	}

	// Пробуем распарсить как относительное время
	if d, err := parseRelative(s); err == nil {
		return time.Now().Add(-d), nil
	}

	// Пробуем распарсить как абсолютное время
	if t, err := parseAbsolute(s); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("unable to parse time: %q", s)
}

// parseRelative парсит относительное время (1h, 30m, 24h).
func parseRelative(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// parseAbsolute парсит абсолютное время в различных форматах.
func parseAbsolute(s string) (time.Time, error) {
	formats := []string{
		// Дата + время
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		// Только дата
		"2006-01-02",
		// ISO8601
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05-07:00",
		// RFC3339
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse absolute time: %q", s)
}
