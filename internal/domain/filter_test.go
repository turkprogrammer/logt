// Package domain тестирует фильтрацию по времени и JSON Path.
package domain

import (
	"testing"
	"time"

	"github.com/turkprogrammer/logt/internal/domain/jsonpath"
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

func TestGetFilteredByJson(t *testing.T) {
	rb := NewRingBuffer(100)

	// Добавляем JSON строки с разными уровнями
	lines := []LogLine{
		{
			Content: `{"level": "error", "message": "connection failed"}`,
			IsJSON:  true,
			Parsed:  map[string]any{"level": "error", "message": "connection failed"},
		},
		{
			Content: `{"level": "info", "message": "started"}`,
			IsJSON:  true,
			Parsed:  map[string]any{"level": "info", "message": "started"},
		},
		{
			Content: `{"level": "error", "message": "timeout occurred"}`,
			IsJSON:  true,
			Parsed:  map[string]any{"level": "error", "message": "timeout occurred"},
		},
		{
			Content: `{"level": "debug", "message": "verbose"}`,
			IsJSON:  true,
			Parsed:  map[string]any{"level": "debug", "message": "verbose"},
		},
	}

	for _, line := range lines {
		rb.Add(line)
	}

	// Фильтр: только error
	filter := &jsonpath.Filter{
		Path:     "level",
		Operator: jsonpath.OpEquals,
		Value:    "error",
	}

	filtered := rb.GetFilteredByJson(filter)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 error lines, got %d", len(filtered))
	}

	// Проверяем, что все отфильтрованные строки — error
	for _, line := range filtered {
		data := line.Parsed.(map[string]any)
		if data["level"] != "error" {
			t.Errorf("Expected level=error, got %v", data["level"])
		}
	}
}

func TestGetFilteredCombined_JsonAndTime(t *testing.T) {
	rb := NewRingBuffer(100)
	now := time.Now()

	// Добавляем JSON строки с разными временными метками и уровнями
	lines := []LogLine{
		{
			Timestamp: now.Add(-2 * time.Hour),
			Content:   `{"level": "error", "message": "old error"}`,
			IsJSON:    true,
			Parsed:    map[string]any{"level": "error", "message": "old error"},
		},
		{
			Timestamp: now.Add(-1 * time.Hour),
			Content:   `{"level": "error", "message": "recent error"}`,
			IsJSON:    true,
			Parsed:    map[string]any{"level": "error", "message": "recent error"},
		},
		{
			Timestamp: now.Add(-30 * time.Minute),
			Content:   `{"level": "info", "message": "very recent info"}`,
			IsJSON:    true,
			Parsed:    map[string]any{"level": "info", "message": "very recent info"},
		},
	}

	for _, line := range lines {
		rb.Add(line)
	}

	// Комбинированный фильтр: JSON Path + время
	since := now.Add(-90 * time.Minute)
	jsonFilter := &jsonpath.Filter{
		Path:     "level",
		Operator: jsonpath.OpEquals,
		Value:    "error",
	}

	filtered := rb.GetFilteredCombined("", nil, &since, nil, jsonFilter)

	if len(filtered) != 1 {
		t.Errorf("Expected 1 line, got %d", len(filtered))
	}

	if filtered[0].Content != `{"level": "error", "message": "recent error"}` {
		t.Errorf("Expected 'recent error', got %q", filtered[0].Content)
	}
}

func TestGetFilteredByJson_NonJsonLines(t *testing.T) {
	rb := NewRingBuffer(100)

	// Добавляем смешанные строки (JSON + plain text)
	lines := []LogLine{
		{
			Content: `{"level": "error", "message": "json error"}`,
			IsJSON:  true,
			Parsed:  map[string]any{"level": "error", "message": "json error"},
		},
		{
			Content: "ERROR: plain text error",
			IsJSON:  false,
			Parsed:  nil,
		},
		{
			Content: `{"level": "info", "message": "json info"}`,
			IsJSON:  true,
			Parsed:  map[string]any{"level": "info", "message": "json info"},
		},
	}

	for _, line := range lines {
		rb.Add(line)
	}

	// Фильтр: только error (должен пропустить plain text)
	filter := &jsonpath.Filter{
		Path:     "level",
		Operator: jsonpath.OpEquals,
		Value:    "error",
	}

	filtered := rb.GetFilteredByJson(filter)

	if len(filtered) != 1 {
		t.Errorf("Expected 1 JSON error line, got %d", len(filtered))
	}
}

func TestGetFilteredByJson_NestedPath(t *testing.T) {
	rb := NewRingBuffer(100)

	// Добавляем JSON строки с вложенными полями
	lines := []LogLine{
		{
			Content: `{"user": {"name": "admin"}, "action": "login"}`,
			IsJSON:  true,
			Parsed:  map[string]any{"user": map[string]any{"name": "admin"}, "action": "login"},
		},
		{
			Content: `{"user": {"name": "guest"}, "action": "login"}`,
			IsJSON:  true,
			Parsed:  map[string]any{"user": map[string]any{"name": "guest"}, "action": "login"},
		},
	}

	for _, line := range lines {
		rb.Add(line)
	}

	// Фильтр по вложенному пути
	filter := &jsonpath.Filter{
		Path:     "user.name",
		Operator: jsonpath.OpEquals,
		Value:    "admin",
	}

	filtered := rb.GetFilteredByJson(filter)

	if len(filtered) != 1 {
		t.Errorf("Expected 1 admin line, got %d", len(filtered))
	}
}
