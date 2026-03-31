// Package domain тестирует парсинг времени для фильтрации.
package domain

import (
	"testing"
	"time"
)

func TestParseSince_Relative(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValid bool
	}{
		{"1 hour", "1h", true},
		{"30 minutes", "30m", true},
		{"1 day", "24h", true},
		{"1 minute", "1m", true},
		{"invalid", "invalid", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSince(tt.input)
			if tt.wantValid {
				if err != nil {
					t.Errorf("ParseSince(%q) unexpected error: %v", tt.input, err)
				}
				// Проверяем, что время в прошлом (для относительного времени)
				if result.After(time.Now()) {
					t.Errorf("ParseSince(%q) = %v, should be in the past", tt.input, result)
				}
			} else {
				if err == nil {
					t.Errorf("ParseSince(%q) expected error, got nil", tt.input)
				}
			}
		})
	}
}

func TestParseSince_Absolute(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValid bool
	}{
		{"YYYY-MM-DD HH:MM", "2024-01-15 10:00", true},
		{"YYYY-MM-DD", "2024-01-15", true},
		{"invalid date", "2024-99-99", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSince(tt.input)
			if tt.wantValid {
				if err != nil {
					t.Errorf("ParseSince(%q) unexpected error: %v", tt.input, err)
				}
				// Для абсолютного времени проверяем, что оно не zero
				if result.IsZero() && tt.input != "" {
					t.Errorf("ParseSince(%q) = zero time, expected valid time", tt.input)
				}
			} else {
				if err == nil {
					t.Errorf("ParseSince(%q) expected error, got nil", tt.input)
				}
			}
		})
	}
}

func TestParseSince_ISO8601(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValid bool
	}{
		{"ISO8601", "2024-01-15T10:00:00", true},
		{"ISO8601 with Z", "2024-01-15T10:00:00Z", true},
		{"ISO8601 with offset", "2024-01-15T10:00:00+03:00", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSince(tt.input)
			if tt.wantValid {
				if err != nil {
					t.Errorf("ParseSince(%q) unexpected error: %v", tt.input, err)
				}
				if result.IsZero() {
					t.Errorf("ParseSince(%q) = zero time, expected valid time", tt.input)
				}
			} else {
				if err == nil {
					t.Errorf("ParseSince(%q) expected error, got nil", tt.input)
				}
			}
		})
	}
}

func TestParseSince_Invalid(t *testing.T) {
	invalidInputs := []string{
		"not-a-time",
		"2024-99-99",
		"15.01.2024", // неправильный формат
	}

	for _, input := range invalidInputs {
		t.Run(input, func(t *testing.T) {
			_, err := ParseSince(input)
			if err == nil {
				t.Errorf("ParseSince(%q) expected error, got nil", input)
			}
		})
	}
}

func TestParseSince_Empty(t *testing.T) {
	_, err := ParseSince("")
	if err == nil {
		t.Error("ParseSince(\"\") expected error, got nil")
	}
}
