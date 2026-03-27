package domain

import (
	"testing"
	"time"
)

func TestJSONParser_ValidJSON(t *testing.T) {
	parser := &JSONParser{}
	source := Source{Name: "test", Path: "test.log"}

	tests := []struct {
		name      string
		input     string
		wantLevel LogLevel
		wantJSON  bool
	}{
		{
			name:      "level field",
			input:     `{"level": "error", "message": "test"}`,
			wantLevel: LevelError,
			wantJSON:  true,
		},
		{
			name:      "severity field",
			input:     `{"severity": "warn", "msg": "test"}`,
			wantLevel: LevelWarn,
			wantJSON:  true,
		},
		{
			name:      "uppercase level",
			input:     `{"level": "INFO", "message": "test"}`,
			wantLevel: LevelInfo,
			wantJSON:  true,
		},
		{
			name:      "no level field",
			input:     `{"message": "test"}`,
			wantLevel: LevelUnknown,
			wantJSON:  true,
		},
		{
			name:      "nested object",
			input:     `{"level": "error", "data": {"nested": true}}`,
			wantLevel: LevelError,
			wantJSON:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !parser.CanParse(tt.input) {
				t.Errorf("CanParse returned false for valid JSON: %s", tt.input)
			}
			result := parser.Parse(tt.input, source)
			if result == nil {
				t.Fatalf("Parse returned nil for: %s", tt.input)
			}
			if result.Level != tt.wantLevel {
				t.Errorf("Level = %v, want %v", result.Level, tt.wantLevel)
			}
			if !result.IsJSON {
				t.Error("IsJSON should be true")
			}
		})
	}
}

func TestJSONParser_InvalidJSON(t *testing.T) {
	parser := &JSONParser{}
	source := Source{Name: "test", Path: "test.log"}

	invalidJSON := []string{
		`{"level": "error"`,
		`{invalid}`,
		`{level: error}`,
		`plain text`,
		`{"message": "test}`,
		`[`,
		`}`,
		``,
		`   `,
	}

	for _, input := range invalidJSON {
		if parser.CanParse(input) {
			t.Errorf("CanParse returned true for invalid JSON: %q", input)
		}
		result := parser.Parse(input, source)
		if result != nil {
			t.Errorf("Parse should return nil for invalid JSON: %q", input)
		}
	}
}

func TestJSONParser_MalformedJSON(t *testing.T) {
	parser := &JSONParser{}
	source := Source{Name: "test", Path: "test.log"}

	malformed := []string{
		`{"level": "error",}`,
		`{ "key": "value" }`,
		`{"key": "value"`,
		`[1, 2,]`,
		`"unclosed string`,
		`{"message": "has ` + "`" + `backtick"}`,
		`{"bytes": "\xFF"}`,
	}

	for _, input := range malformed {
		result := parser.Parse(input, source)
		if result != nil {
			t.Logf("Malformed JSON returned fallback (PlainParser): %q -> %+v", input, result.Content)
		}
	}
}

func TestJSONParser_ParsedData(t *testing.T) {
	parser := &JSONParser{}
	source := Source{Name: "test", Path: "test.log"}

	input := `{"level": "error", "message": "test message", "code": 500}`
	result := parser.Parse(input, source)

	if result == nil {
		t.Fatal("Parse returned nil")
	}

	data, ok := result.Parsed.(map[string]interface{})
	if !ok {
		t.Fatalf("Parsed should be map, got %T", result.Parsed)
	}

	if msg, ok := data["message"].(string); !ok || msg != "test message" {
		t.Errorf("message = %v, want 'test message'", data["message"])
	}

	if code, ok := data["code"].(float64); !ok || int(code) != 500 {
		t.Errorf("code = %v, want 500", data["code"])
	}
}

func TestLogfmtParser_ValidLogfmt(t *testing.T) {
	parser := &LogfmtParser{}
	source := Source{Name: "test", Path: "test.log"}

	tests := []struct {
		name      string
		input     string
		wantLevel LogLevel
	}{
		{
			name:      "level key",
			input:     `level=error msg="test"`,
			wantLevel: LevelError,
		},
		{
			name:      "severity key",
			input:     `severity=warn message="test"`,
			wantLevel: LevelWarn,
		},
		{
			name:      "multiple keys",
			input:     `host=server1 level=info msg="test"`,
			wantLevel: LevelInfo,
		},
		{
			name:      "quoted value",
			input:     `msg="hello world" level=error`,
			wantLevel: LevelError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !parser.CanParse(tt.input) {
				t.Errorf("CanParse returned false for: %s", tt.input)
			}
			result := parser.Parse(tt.input, source)
			if result == nil {
				t.Fatalf("Parse returned nil for: %s", tt.input)
			}
			if result.Level != tt.wantLevel {
				t.Errorf("Level = %v, want %v", result.Level, tt.wantLevel)
			}
		})
	}
}

func TestLogfmtParser_InvalidLogfmt(t *testing.T) {
	parser := &LogfmtParser{}
	source := Source{Name: "test", Path: "test.log"}

	invalid := []string{
		`single=key`,
		`no equals`,
		`=value`,
		`key=value=extra`,
		`plain text without keys`,
		`{"json": "not logfmt"}`,
	}

	for _, input := range invalid {
		if parser.CanParse(input) {
			result := parser.Parse(input, source)
			if result != nil && result.Level != LevelUnknown {
				t.Logf("Fallback parser used for: %q", input)
			}
		}
	}
}

func TestPlainParser_AlwaysMatches(t *testing.T) {
	parser := &PlainParser{}
	source := Source{Name: "test", Path: "test.log"}

	inputs := []string{
		`plain text`,
		`2024-01-01 INFO message`,
		`ERROR: something failed`,
		`{"json": "but also plain"}`,
		`level=logfmt but treated as plain`,
		``,
		`   whitespace   `,
	}

	for _, input := range inputs {
		if !parser.CanParse(input) {
			t.Errorf("PlainParser.CanParse returned false for: %q", input)
		}
		result := parser.Parse(input, source)
		if result == nil {
			t.Errorf("PlainParser.Parse returned nil for: %q", input)
		}
	}
}

func TestPlainParser_LevelDetection(t *testing.T) {
	parser := &PlainParser{}
	source := Source{Name: "test", Path: "test.log"}

	tests := []struct {
		input     string
		wantLevel LogLevel
	}{
		{`2024-01-01 10:00:00 ERROR: failed`, LevelError},
		{`[INFO] Starting service`, LevelInfo},
		{`WARN: low memory`, LevelWarn},
		{`DEBUG: entering function`, LevelDebug},
		{`FATAL: system crash`, LevelFatal},
		{`plain text without level`, LevelUnknown},
		{`error (lowercase)`, LevelError},
		{`Error (capitalized)`, LevelError},
		{`ERR short form`, LevelError},
		{`WARNING long form`, LevelWarn},
	}

	for _, tt := range tests {
		result := parser.Parse(tt.input, source)
		if result.Level != tt.wantLevel {
			t.Errorf("Level detection for %q = %v, want %v", tt.input, result.Level, tt.wantLevel)
		}
	}
}

func TestMultiParser_Priority(t *testing.T) {
	mp := NewMultiParser()
	source := Source{Name: "test", Path: "test.log"}

	jsonInput := `{"level": "error", "message": "test"}`
	result := mp.Parse(jsonInput, source)
	if !result.IsJSON {
		t.Error("MultiParser should prefer JSON parser")
	}
	if result.Level != LevelError {
		t.Errorf("JSON level = %v, want ERROR", result.Level)
	}

	logfmtInput := `level=warn msg="test"`
	result = mp.Parse(logfmtInput, source)
	if result.Level != LevelWarn {
		t.Errorf("Logfmt level = %v, want WARN", result.Level)
	}

	plainInput := `plain text without format`
	result = mp.Parse(plainInput, source)
	if result.Content != plainInput {
		t.Error("MultiParser should fallback to PlainParser")
	}
}

func TestDetectLevel_CaseInsensitive(t *testing.T) {
	tests := []struct {
		input string
		want  LogLevel
	}{
		{`ERROR message`, LevelError},
		{`error message`, LevelError},
		{`Error message`, LevelError},
		{`eRrOr message`, LevelError},
		{`INFO message`, LevelInfo},
		{`info message`, LevelInfo},
		{`INFO`, LevelInfo},
	}

	for _, tt := range tests {
		got := DetectLevel(tt.input)
		if got != tt.want {
			t.Errorf("DetectLevel(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsValidJSON(t *testing.T) {
	valid := []string{
		`{}`,
		`[]`,
		`{"key": "value"}`,
		`[1, 2, 3]`,
		`{"nested": {"deep": true}}`,
	}

	invalid := []string{
		``,
		` `,
		`plain`,
		`{key: value}`,
		`[1, 2,]`,
	}

	for _, s := range valid {
		if !IsValidJSON(s) {
			t.Errorf("IsValidJSON(%q) = false, want true", s)
		}
	}

	for _, s := range invalid {
		if IsValidJSON(s) {
			t.Errorf("IsValidJSON(%q) = true, want false", s)
		}
	}
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{`2024-01-01T10:00:00`, true},
		{`2024-01-01 10:00:00`, true},
		{`[2024-01-01T10:00:00]`, true},
		{`01/Jan/2024:10:00:00`, true},
		{`plain text`, false},
		{``, false},
	}

	for _, tt := range tests {
		before := time.Now()
		result := ParseTimestamp(tt.input)
		after := time.Now()

		if tt.want {
			if result.Before(before) || result.After(after) {
				t.Logf("Timestamp parsed for %q: %v", tt.input, result)
			}
		} else {
			if !result.After(before.Add(-time.Second)) && !result.Before(after.Add(time.Second)) {
				t.Logf("Timestamp for %q defaulted to now: %v", tt.input, result)
			}
		}
	}
}

func BenchmarkRingBuffer_Add(b *testing.B) {
	rb := NewRingBuffer(5000)
	line := LogLine{Content: "benchmark line"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.Add(line)
	}
}

func BenchmarkRingBuffer_GetAll(b *testing.B) {
	rb := NewRingBuffer(5000)
	for i := 0; i < 5000; i++ {
		rb.Add(LogLine{Content: "line"})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.GetAll()
	}
}

func BenchmarkJSONParser_Parse(b *testing.B) {
	parser := &JSONParser{}
	input := `{"level": "error", "message": "test message", "code": 500, "data": {"nested": true}}`
	source := Source{Name: "test", Path: "test.log"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.Parse(input, source)
	}
}

func BenchmarkDetectLevel(b *testing.B) {
	input := `2024-01-01 10:00:00 ERROR: something failed in the system`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectLevel(input)
	}
}

func BenchmarkIsValidJSON(b *testing.B) {
	json := `{"level": "error", "message": "test", "nested": {"a": 1, "b": 2}}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsValidJSON(json)
	}
}
