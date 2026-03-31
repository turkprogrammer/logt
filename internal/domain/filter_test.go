// Package domain тестирует фильтрацию по времени.
package domain

import (
	"testing"
	"time"
)

func TestGetFilteredWithTime_Since(t *testing.T) {
	rb := NewRingBuffer(100)
	now := time.Now()

	// Добавляем строки с разными временными метками
	lines := []LogLine{
		{Timestamp: now.Add(-2 * time.Hour), Content: "old"},
		{Timestamp: now.Add(-1 * time.Hour), Content: "recent"},
		{Timestamp: now.Add(-30 * time.Minute), Content: "very recent"},
	}

	for _, line := range lines {
		rb.Add(line)
	}

	// Фильтр: только логи за последний час
	since := now.Add(-90 * time.Minute)
	filtered := rb.GetFilteredWithTime("", nil, &since, nil)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(filtered))
	}

	// Проверяем, что "old" отфильтрован
	for _, line := range filtered {
		if line.Content == "old" {
			t.Error("Old line should be filtered out")
		}
	}
}

func TestGetFilteredWithTime_Until(t *testing.T) {
	rb := NewRingBuffer(100)
	now := time.Now()

	lines := []LogLine{
		{Timestamp: now.Add(-2 * time.Hour), Content: "old"},
		{Timestamp: now.Add(-1 * time.Hour), Content: "recent"},
		{Timestamp: now.Add(-30 * time.Minute), Content: "very recent"},
	}

	for _, line := range lines {
		rb.Add(line)
	}

	// Фильтр: только логи до 1 часа назад
	until := now.Add(-90 * time.Minute)
	filtered := rb.GetFilteredWithTime("", nil, nil, &until)

	if len(filtered) != 1 {
		t.Errorf("Expected 1 line, got %d", len(filtered))
	}

	if filtered[0].Content != "old" {
		t.Errorf("Expected 'old' line, got %q", filtered[0].Content)
	}
}

func TestGetFilteredWithTime_Combined(t *testing.T) {
	rb := NewRingBuffer(100)
	now := time.Now()

	lines := []LogLine{
		{Timestamp: now.Add(-2 * time.Hour), Content: "old error"},
		{Timestamp: now.Add(-1 * time.Hour), Content: "recent error"},
		{Timestamp: now.Add(-30 * time.Minute), Content: "very recent info"},
	}

	for _, line := range lines {
		rb.Add(line)
	}

	// Комбинированный фильтр: время + текст
	since := now.Add(-90 * time.Minute)
	until := now.Add(-15 * time.Minute)
	filtered := rb.GetFilteredWithTime("error", nil, &since, &until)

	if len(filtered) != 1 {
		t.Errorf("Expected 1 line, got %d", len(filtered))
	}

	if filtered[0].Content != "recent error" {
		t.Errorf("Expected 'recent error', got %q", filtered[0].Content)
	}
}

func TestGetFilteredWithTime_NoFilters(t *testing.T) {
	rb := NewRingBuffer(100)
	now := time.Now()

	lines := []LogLine{
		{Timestamp: now.Add(-2 * time.Hour), Content: "old"},
		{Timestamp: now.Add(-1 * time.Hour), Content: "recent"},
	}

	for _, line := range lines {
		rb.Add(line)
	}

	// Без фильтров
	filtered := rb.GetFilteredWithTime("", nil, nil, nil)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(filtered))
	}
}

func TestGetFilteredWithTime_EmptyBuffer(t *testing.T) {
	rb := NewRingBuffer(100)
	now := time.Now()

	since := now.Add(-1 * time.Hour)
	filtered := rb.GetFilteredWithTime("", nil, &since, nil)

	if len(filtered) != 0 {
		t.Errorf("Expected 0 lines for empty buffer, got %d", len(filtered))
	}
}
