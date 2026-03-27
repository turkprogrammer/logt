package domain

import (
	"strings"
	"testing"
)

func TestFuzzyMatch_Basic(t *testing.T) {
	tests := []struct {
		text    string
		pattern string
		want    bool
	}{
		{"hello world", "hello", true},
		{"hello world", "world", true},
		{"hello world", "hello world", true},
		{"hello world", "HELLO", true},
		{"hello world", "xyz", false},
		{"", "", true},
		{"", "a", false},
		{"text", "", true},
		{"test", "tst", true},
	}

	for _, tt := range tests {
		got := FuzzyMatch(tt.text, tt.pattern)
		if got != tt.want {
			t.Errorf("FuzzyMatch(%q, %q) = %v, want %v", tt.text, tt.pattern, got, tt.want)
		}
	}
}

func TestFuzzyMatch_Contained(t *testing.T) {
	tests := []struct {
		text    string
		pattern string
		want    bool
	}{
		{"error: connection failed", "error", true},
		{"error: connection failed", "failed", true},
		{"error: connection failed", "connection", true},
		{"[INFO] Service started", "INFO", true},
		{"[INFO] Service started", "service", true},
		{"2024-01-01 ERROR", "2024", true},
	}

	for _, tt := range tests {
		got := fuzzyMatchRecursive(strings.ToLower(tt.text), strings.ToLower(tt.pattern))
		if got != tt.want {
			t.Errorf("fuzzyMatch(%q, %q) = %v, want %v", tt.text, tt.pattern, got, tt.want)
		}
	}
}

func TestFuzzyMatch_ComplexPatterns(t *testing.T) {
	tests := []struct {
		text    string
		pattern string
		want    bool
	}{
		{"docker logs output", "dock", true},
		{"kubernetes pod info", "k8s", false},
		{"json log entry", "json", true},
		{"plain text log", "plain", true},
		{"timestampped message", "stamp", true},
	}

	for _, tt := range tests {
		got := fuzzyMatchRecursive(strings.ToLower(tt.text), strings.ToLower(tt.pattern))
		if got != tt.want {
			t.Errorf("fuzzyMatch(%q, %q) = %v, want %v", tt.text, tt.pattern, got, tt.want)
		}
	}
}

func TestHighlightMatches_Basic(t *testing.T) {
	tests := []struct {
		text      string
		pattern   string
		wantHasHL bool
	}{
		{"hello world", "hello", true},
		{"hello world", "world", true},
		{"hello world", "o", true},
		{"test", "xyz", false},
		{"", "", false},
	}

	for _, tt := range tests {
		result := HighlightMatches(tt.text, tt.pattern)

		if tt.wantHasHL {
			if !strings.Contains(result, "\x02") || !strings.Contains(result, "\x03") {
				t.Errorf("HighlightMatches(%q, %q) missing highlights: %q", tt.text, tt.pattern, result)
			}
		}
	}
}

func TestHighlightMatches_MultipleMatches(t *testing.T) {
	text := "error occurred in error handler"
	pattern := "error"

	result := HighlightMatches(text, pattern)

	count := strings.Count(result, "\x02")
	if count != 2 {
		t.Errorf("Expected 2 highlights, got %d in %q", count, result)
	}
}

func TestHighlightMatches_CaseInsensitive(t *testing.T) {
	text := "ERROR error Error"
	pattern := "error"

	result := HighlightMatches(text, pattern)

	count := strings.Count(result, "\x02")
	if count != 3 {
		t.Errorf("Expected 3 highlights (case insensitive), got %d", count)
	}
}

func TestHighlightMatches_EmptyPattern(t *testing.T) {
	text := "hello world"
	pattern := ""

	result := HighlightMatches(text, pattern)

	if result != text {
		t.Errorf("Empty pattern should return original text, got %q", result)
	}
}

func TestHighlightMatches_NoMatch(t *testing.T) {
	text := "hello world"
	pattern := "xyz"

	result := HighlightMatches(text, pattern)

	if result != text {
		t.Errorf("No match should return original text, got %q", result)
	}
}

func TestHighlightMatches_RealWorldLogs(t *testing.T) {
	logs := []string{
		`2024-01-01 10:00:00 ERROR connection_timeout db=localhost`,
		`{"level": "error", "message": "database connection failed"}`,
		`[WARN] High memory usage: 95%`,
	}

	patterns := []string{"ERROR", "error", "failed", "WARN", "memory"}

	for _, log := range logs {
		for _, pattern := range patterns {
			result := HighlightMatches(log, pattern)
			hasHL := strings.Contains(result, "\x02")
			hasPattern := strings.Contains(strings.ToLower(log), strings.ToLower(pattern))

			if hasPattern != hasHL {
				t.Errorf("HighlightMatches mismatch for log=%q pattern=%q: hasPattern=%v hasHL=%v",
					log, pattern, hasPattern, hasHL)
			}
		}
	}
}

func TestGetFiltered_WithPattern(t *testing.T) {
	rb := NewRingBuffer(100)

	logs := []string{
		`[INFO] Service started`,
		`[ERROR] Connection failed`,
		`[INFO] Request received`,
		`[WARN] High latency detected`,
		`[ERROR] Timeout occurred`,
	}

	for _, log := range logs {
		rb.Add(LogLine{Content: log})
	}

	tests := []struct {
		pattern string
		want    int
	}{
		{"ERROR", 2},
		{"INFO", 2},
		{"WARN", 1},
		{"timeout", 1},
		{"nonexistent", 0},
		{"", 5},
	}

	for _, tt := range tests {
		filtered := rb.GetFiltered(tt.pattern, nil)
		if len(filtered) != tt.want {
			t.Errorf("GetFiltered(%q) = %d, want %d", tt.pattern, len(filtered), tt.want)
		}
	}
}

func TestGetFiltered_CaseInsensitive(t *testing.T) {
	rb := NewRingBuffer(100)

	rb.Add(LogLine{Content: "ERROR message"})
	rb.Add(LogLine{Content: "error message"})
	rb.Add(LogLine{Content: "Error message"})
	rb.Add(LogLine{Content: "info message"})

	filtered := rb.GetFiltered("ERROR", nil)
	if len(filtered) != 3 {
		t.Errorf("Expected 3 ERROR lines (case insensitive), got %d", len(filtered))
	}
}

func TestGetFiltered_CombinedWithSource(t *testing.T) {
	rb := NewRingBuffer(100)

	rb.Add(LogLine{Content: "error from file1", Source: Source{Path: "file1.log"}})
	rb.Add(LogLine{Content: "info from file1", Source: Source{Path: "file1.log"}})
	rb.Add(LogLine{Content: "error from file2", Source: Source{Path: "file2.log"}})
	rb.Add(LogLine{Content: "info from file2", Source: Source{Path: "file2.log"}})

	sources := map[string]bool{"file1.log": true}
	filtered := rb.GetFiltered("error", sources)

	if len(filtered) != 1 {
		t.Errorf("Expected 1 error from file1, got %d", len(filtered))
	}

	filtered = rb.GetFiltered("", sources)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 lines from file1, got %d", len(filtered))
	}
}

func TestSearchMatches_RealWorld(t *testing.T) {
	rb := NewRingBuffer(5000)
	source := Source{Name: "app.log", Path: "app.log"}

	logLines := []string{
		`2024-01-01 10:00:00 INFO Starting application`,
		`2024-01-01 10:00:01 DEBUG Loading configuration`,
		`2024-01-01 10:00:02 INFO Database connected`,
		`2024-01-01 10:00:03 WARN Connection pool low`,
		`2024-01-01 10:00:04 ERROR Failed to process request`,
		`2024-01-01 10:00:05 INFO Request completed`,
		`{"level": "error", "message": "validation failed", "field": "email"}`,
		`{"level": "info", "message": "user logged in", "user_id": 123}`,
	}

	for _, line := range logLines {
		rb.Add(LogLine{Content: line, Source: source, IsJSON: strings.HasPrefix(line, "{")})
	}

	tests := []struct {
		pattern string
		want    int
	}{
		{"INFO", 4},
		{"ERROR", 2},
		{"WARN", 1},
		{"DEBUG", 1},
		{"database", 1},
		{"user", 1},
		{"failed", 2},
		{"connection", 1},
		{"xyz", 0},
	}

	for _, tt := range tests {
		filtered := rb.GetFiltered(tt.pattern, nil)
		if len(filtered) != tt.want {
			t.Errorf("GetFiltered(%q) = %d, want %d", tt.pattern, len(filtered), tt.want)
		}
	}
}

func BenchmarkFuzzyMatch(b *testing.B) {
	text := "2024-01-01 10:00:00 ERROR connection_timeout db=localhost timeout"
	pattern := "error"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fuzzyMatchRecursive(strings.ToLower(text), strings.ToLower(pattern))
	}
}

func BenchmarkHighlightMatches(b *testing.B) {
	text := `2024-01-01 10:00:00 ERROR connection_timeout db=localhost timeout occurred`
	pattern := "error"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HighlightMatches(text, pattern)
	}
}

func BenchmarkGetFiltered(b *testing.B) {
	rb := NewRingBuffer(5000)
	for i := 0; i < 5000; i++ {
		rb.Add(LogLine{Content: "INFO log entry"})
	}
	for i := 0; i < 1000; i++ {
		rb.Add(LogLine{Content: "ERROR error occurred"})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.GetFiltered("ERROR", nil)
	}
}
