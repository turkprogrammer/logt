package domain

import (
	"testing"
	"time"

	"github.com/turkprogrammer/logt/internal/domain/jsonpath"
)

func TestGetFilteredCombined_JsonAndTime(t *testing.T) {
	rb := NewRingBuffer(100)
	now := time.Now()

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

	since := now.Add(-90 * time.Minute)
	jsonFilter := &jsonpath.Filter{
		Path:     "level",
		Operator: jsonpath.OpEquals,
		Value:    "error",
	}

	filtered := rb.GetFilteredCombined(FilterOptions{
		Text:       "",
		Since:      &since,
		JSONFilter: jsonFilter,
	})

	if len(filtered) != 1 {
		t.Errorf("Expected 1 line, got %d", len(filtered))
	}

	if filtered[0].Content != `{"level": "error", "message": "recent error"}` {
		t.Errorf("Expected 'recent error', got %q", filtered[0].Content)
	}
}
