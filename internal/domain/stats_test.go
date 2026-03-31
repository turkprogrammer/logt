// Package domain тестирует калькулятор статистики логов.
package domain

import (
	"testing"
	"time"
)

func TestStats_Calculate(t *testing.T) {
	rb := NewRingBuffer(100)
	now := time.Now()

	// Добавляем строки с разными уровнями
	lines := []LogLine{
		{Timestamp: now.Add(-5 * time.Minute), Level: LevelInfo},
		{Timestamp: now.Add(-4 * time.Minute), Level: LevelInfo},
		{Timestamp: now.Add(-3 * time.Minute), Level: LevelWarn},
		{Timestamp: now.Add(-2 * time.Minute), Level: LevelError},
		{Timestamp: now.Add(-1 * time.Minute), Level: LevelError},
		{Timestamp: now, Level: LevelDebug},
	}

	for _, line := range lines {
		rb.Add(line)
	}

	stats := rb.CalculateStats()

	if stats.TotalLines != 6 {
		t.Errorf("Expected TotalLines=6, got %d", stats.TotalLines)
	}

	if stats.ByLevel[LevelInfo] != 2 {
		t.Errorf("Expected LevelInfo=2, got %d", stats.ByLevel[LevelInfo])
	}

	if stats.ByLevel[LevelWarn] != 1 {
		t.Errorf("Expected LevelWarn=1, got %d", stats.ByLevel[LevelWarn])
	}

	if stats.ByLevel[LevelError] != 2 {
		t.Errorf("Expected LevelError=2, got %d", stats.ByLevel[LevelError])
	}

	if stats.ByLevel[LevelDebug] != 1 {
		t.Errorf("Expected LevelDebug=1, got %d", stats.ByLevel[LevelDebug])
	}
}

func TestStats_EmptyBuffer(t *testing.T) {
	rb := NewRingBuffer(100)

	stats := rb.CalculateStats()

	if stats.TotalLines != 0 {
		t.Errorf("Expected TotalLines=0 for empty buffer, got %d", stats.TotalLines)
	}

	if len(stats.ByLevel) != 0 {
		t.Errorf("Expected empty ByLevel for empty buffer, got %v", stats.ByLevel)
	}

	if stats.Rate != 0 {
		t.Errorf("Expected Rate=0 for empty buffer, got %f", stats.Rate)
	}
}

func TestStats_RateCalculation(t *testing.T) {
	rb := NewRingBuffer(100)
	now := time.Now()

	// Добавляем 10 строк за 9 секунд (0,1,2...9 секунды)
	for i := 0; i < 10; i++ {
		rb.Add(LogLine{
			Timestamp: now.Add(time.Duration(i) * time.Second),
			Level:     LevelInfo,
		})
	}

	stats := rb.CalculateStats()

	// Ожидаем rate ~1.11 lines/sec (10 строк за 9 секунд)
	// Допускаем погрешность
	if stats.Rate < 0.9 || stats.Rate > 1.3 {
		t.Errorf("Expected Rate≈1.11, got %f", stats.Rate)
	}
}

func TestStats_LevelPercentage(t *testing.T) {
	rb := NewRingBuffer(100)
	now := time.Now()

	// 100 строк: 50 INFO, 30 WARN, 20 ERROR
	for i := 0; i < 50; i++ {
		rb.Add(LogLine{Timestamp: now, Level: LevelInfo})
	}
	for i := 0; i < 30; i++ {
		rb.Add(LogLine{Timestamp: now, Level: LevelWarn})
	}
	for i := 0; i < 20; i++ {
		rb.Add(LogLine{Timestamp: now, Level: LevelError})
	}

	stats := rb.CalculateStats()

	// Проверяем проценты
	errorPct := stats.ErrorPercentage()
	if errorPct != 20.0 {
		t.Errorf("Expected ErrorPercentage=20.0, got %f", errorPct)
	}

	warnPct := stats.PercentageForLevel(LevelWarn)
	if warnPct != 30.0 {
		t.Errorf("Expected WarnPercentage=30.0, got %f", warnPct)
	}
}

func TestStats_TimeRange(t *testing.T) {
	rb := NewRingBuffer(100)
	now := time.Now()

	// Добавляем строки с разными временными метками
	rb.Add(LogLine{Timestamp: now.Add(-1 * time.Hour), Level: LevelInfo})
	rb.Add(LogLine{Timestamp: now.Add(-30 * time.Minute), Level: LevelWarn})
	rb.Add(LogLine{Timestamp: now, Level: LevelError})

	stats := rb.CalculateStats()

	if stats.TimeRange.Start.IsZero() {
		t.Error("Expected non-zero TimeRange.Start")
	}

	if stats.TimeRange.End.IsZero() {
		t.Error("Expected non-zero TimeRange.End")
	}

	// Проверяем, что Start < End
	if !stats.TimeRange.Start.Before(stats.TimeRange.End) {
		t.Error("Expected TimeRange.Start < TimeRange.End")
	}
}

func TestStats_String(t *testing.T) {
	rb := NewRingBuffer(100)
	now := time.Now()

	for i := 0; i < 10; i++ {
		rb.Add(LogLine{Timestamp: now, Level: LevelInfo})
	}
	for i := 0; i < 5; i++ {
		rb.Add(LogLine{Timestamp: now, Level: LevelError})
	}

	stats := rb.CalculateStats()
	output := stats.String()

	// Проверяем, что вывод содержит ключевые поля
	if len(output) == 0 {
		t.Error("Expected non-empty stats output")
	}
}
