// Package domain предоставляет функции для расчёта статистики логов.
package domain

import (
	"fmt"
	"strings"
	"time"
)

// TimeRange представляет временной диапазон логов.
type TimeRange struct {
	Start time.Time // Начало диапазона
	End   time.Time // Конец диапазона
}

// Stats представляет агрегированную статистику логов.
type Stats struct {
	TotalLines int              // Общее количество строк
	ByLevel    map[LogLevel]int // Количество строк по уровням
	TimeRange  TimeRange        // Временной диапазон
	Rate       float64          // Средняя скорость (строк/секунду)
}

// CalculateStats рассчитывает статистику для буфера логов.
// Проход O(n) по всем строкам.
func (rb *RingBuffer) CalculateStats() *Stats {
	lines := rb.GetAll()

	stats := &Stats{
		TotalLines: len(lines),
		ByLevel:    make(map[LogLevel]int),
		TimeRange:  TimeRange{},
		Rate:       0,
	}

	if len(lines) == 0 {
		return stats
	}

	// Инициализируем временной диапазон
	minTime := lines[0].Timestamp
	maxTime := lines[0].Timestamp

	// Считаем уровни и находим временной диапазон
	for _, line := range lines {
		stats.ByLevel[line.Level]++

		if line.Timestamp.Before(minTime) {
			minTime = line.Timestamp
		}
		if line.Timestamp.After(maxTime) {
			maxTime = line.Timestamp
		}
	}

	stats.TimeRange = TimeRange{
		Start: minTime,
		End:   maxTime,
	}

	// Рассчитываем rate (строк в секунду)
	duration := maxTime.Sub(minTime)
	if duration > 0 {
		stats.Rate = float64(len(lines)) / duration.Seconds()
	}

	return stats
}

// ErrorPercentage возвращает процент ошибок от общего количества строк.
func (s *Stats) ErrorPercentage() float64 {
	if s.TotalLines == 0 {
		return 0
	}
	errors := s.ByLevel[LevelError] + s.ByLevel[LevelFatal]
	return float64(errors) * 100.0 / float64(s.TotalLines)
}

// PercentageForLevel возвращает процент для указанного уровня.
func (s *Stats) PercentageForLevel(level LogLevel) float64 {
	if s.TotalLines == 0 {
		return 0
	}
	count := s.ByLevel[level]
	return float64(count) * 100.0 / float64(s.TotalLines)
}

// String возвращает человекочитаемое представление статистики.
func (s *Stats) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Lines: %d\n", s.TotalLines))

	// Процент ошибок
	errorPct := s.ErrorPercentage()
	sb.WriteString(fmt.Sprintf("Errors: %d (%.2f%%)\n", s.ByLevel[LevelError]+s.ByLevel[LevelFatal], errorPct))

	// По уровням
	var levelParts []string
	levels := []LogLevel{LevelError, LevelWarn, LevelInfo, LevelDebug}
	for _, level := range levels {
		if count, ok := s.ByLevel[level]; ok && count > 0 {
			levelParts = append(levelParts, fmt.Sprintf("%s=%d", level, count))
		}
	}
	if len(levelParts) > 0 {
		sb.WriteString(fmt.Sprintf("By level: %s\n", strings.Join(levelParts, ", ")))
	}

	// Временной диапазон
	if !s.TimeRange.Start.IsZero() && !s.TimeRange.End.IsZero() {
		startStr := s.TimeRange.Start.Format("15:04:05")
		endStr := s.TimeRange.End.Format("15:04:05")
		sb.WriteString(fmt.Sprintf("Time range: %s → %s\n", startStr, endStr))
	}

	// Rate
	if s.Rate > 0 {
		sb.WriteString(fmt.Sprintf("Rate: ~%.0f lines/sec\n", s.Rate))
	}

	return sb.String()
}
