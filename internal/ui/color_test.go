package ui

import (
	"testing"
)

func TestParseColorMode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ColorMode
		wantErr bool
	}{
		{"auto", "auto", ColorAuto, false},
		{"always", "always", ColorAlways, false},
		{"never", "never", ColorNever, false},
		{"empty", "", ColorAuto, false},
		{"invalid", "invalid", ColorAuto, true},
		{"case insensitive", "AUTO", ColorAuto, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseColorMode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseColorMode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseColorMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldUseColor(t *testing.T) {
	tests := []struct {
		name       string
		mode       ColorMode
		isTerminal bool
		want       bool
	}{
		{"auto + terminal", ColorAuto, true, true},
		{"auto + not terminal", ColorAuto, false, false},
		{"always + terminal", ColorAlways, true, true},
		{"always + not terminal", ColorAlways, false, true},
		{"never + terminal", ColorNever, true, false},
		{"never + not terminal", ColorNever, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldUseColor(tt.mode, tt.isTerminal)
			if got != tt.want {
				t.Errorf("ShouldUseColor(%v, %v) = %v, want %v", tt.mode, tt.isTerminal, got, tt.want)
			}
		})
	}
}
