package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestExpandPaths тестирует раскрытие glob паттернов.
func TestExpandPaths(t *testing.T) {
	// Создаём временные файлы
	tmpDir := t.TempDir()

	files := []string{"test1.log", "test2.log", "test3.txt"}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	tests := []struct {
		name     string
		pattern  string
		expected int
	}{
		{"single file", filepath.Join(tmpDir, "test1.log"), 1},
		{"glob *.log", filepath.Join(tmpDir, "*.log"), 2},
		{"glob *.txt", filepath.Join(tmpDir, "*.txt"), 1},
		{"glob all", filepath.Join(tmpDir, "*"), 3},
		{"nonexistent", filepath.Join(tmpDir, "nonexistent.*"), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandPaths([]string{tt.pattern})
			if len(result) != tt.expected {
				t.Errorf("Expected %d files, got %d: %v", tt.expected, len(result), result)
			}
		})
	}
}

// TestFileProviderWatch тестирует базовую функциональность FileProvider.
func TestFileProviderWatch(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.log")

	// Создаём файл с начальным содержимым
	initialContent := `{"level":"info","msg":"Test message 1"}
{"level":"error","msg":"Test message 2"}
`
	if err := os.WriteFile(testFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	provider := NewFileProvider()

	if err := provider.Watch([]string{testFile}); err != nil {
		t.Fatalf("Failed to watch file: %v", err)
	}
	defer provider.Close()

	// Ждём немного для чтения существующего содержимого
	time.Sleep(200 * time.Millisecond)

	// Проверяем, что прочитали начальные строки
	count := 0
	timeout := time.After(500 * time.Millisecond)
	for {
		select {
		case _, ok := <-provider.LogChan():
			if !ok {
				t.Fatal("Log channel closed unexpectedly")
			}
			count++
			if count >= 2 {
				goto done
			}
		case <-timeout:
			t.Fatalf("Timeout waiting for log lines, got %d", count)
		}
	}

done:
	t.Logf("Successfully read %d log lines", count)
}

// TestFileProviderRotation тестирует обработку ротации файлов.
func TestFileProviderRotation(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.log")

	// Создаём файл с содержимым
	initialContent := `{"level":"info","msg":"Initial message"}
`
	if err := os.WriteFile(testFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	provider := NewFileProvider()

	if err := provider.Watch([]string{testFile}); err != nil {
		t.Fatalf("Failed to watch file: %v", err)
	}
	defer provider.Close()

	// Ждём чтения начального содержимого
	time.Sleep(200 * time.Millisecond)

	// "Ротируем" файл - перезаписываем с меньшим размером
	rotatedContent := `{"level":"warn","msg":"After rotation"}
{"level":"error","msg":"New error after rotation"}
`
	if err := os.WriteFile(testFile, []byte(rotatedContent), 0644); err != nil {
		t.Fatalf("Failed to rotate test file: %v", err)
	}

	// Ждём обнаружения изменений
	time.Sleep(300 * time.Millisecond)

	// Проверяем, что прочитали новые строки
	foundRotation := false
	timeout := time.After(500 * time.Millisecond)
	for {
		select {
		case logLine, ok := <-provider.LogChan():
			if !ok {
				t.Fatal("Log channel closed unexpectedly")
			}
			// Проверяем, что это новая строка (после ротации)
			if strings.Contains(logLine.Content, "After rotation") || strings.Contains(logLine.Content, "New error") {
				foundRotation = true
				t.Logf("Successfully detected rotation: %s", logLine.Content)
				goto done
			}
		case <-timeout:
			if !foundRotation {
				t.Fatal("Timeout waiting for rotated content")
			}
		}
	}

done:
	if !foundRotation {
		t.Fatal("Did not detect file rotation")
	}
}

// TestMultiProvider тестирует объединение провайдеров.
func TestMultiProvider(t *testing.T) {
	tmpDir := t.TempDir()

	// Создаём два файла
	file1 := filepath.Join(tmpDir, "app1.log")
	file2 := filepath.Join(tmpDir, "app2.log")

	if err := os.WriteFile(file1, []byte(`{"level":"info","msg":"App1 message"}`), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte(`{"level":"error","msg":"App2 message"}`), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	mp := NewMultiProvider()
	defer mp.Close()

	fp1 := NewFileProvider()
	fp2 := NewFileProvider()

	if err := fp1.Watch([]string{file1}); err != nil {
		t.Fatalf("Failed to watch file1: %v", err)
	}
	if err := fp2.Watch([]string{file2}); err != nil {
		t.Fatalf("Failed to watch file2: %v", err)
	}

	mp.AddProvider(fp1)
	mp.AddProvider(fp2)

	// Ждём чтения
	time.Sleep(300 * time.Millisecond)

	// Проверяем источники
	sources := mp.Sources()
	if len(sources) != 2 {
		t.Errorf("Expected 2 sources, got %d", len(sources))
	}

	t.Logf("Sources: %v", sources)
}

// TestStdinProvider тестирует чтение из stdin.
func TestStdinProvider(t *testing.T) {
	// Создаём pipe для эмуляции stdin
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	// Сохраняем оригинальный stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	os.Stdin = reader

	provider := NewStdinProvider()
	if err := provider.Start(); err != nil {
		t.Fatalf("Failed to start provider: %v", err)
	}
	defer provider.Close()

	// Пишем в pipe
	testLine := `{"level":"info","msg":"Test from stdin"}
`
	if _, err := writer.WriteString(testLine); err != nil {
		t.Fatalf("Failed to write to pipe: %v", err)
	}
	writer.Close()

	// Ждём чтения
	select {
	case logLine, ok := <-provider.LogChan():
		if !ok {
			t.Fatal("Log channel closed unexpectedly")
		}
		if logLine.Content != `{"level":"info","msg":"Test from stdin"}` {
			t.Errorf("Unexpected content: %s", logLine.Content)
		}
		t.Logf("Successfully read from stdin: %s", logLine.Content)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timeout waiting for stdin content")
	}
}

// TestToggleSource тестирует переключение источников.
func TestToggleSource(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.log")

	if err := os.WriteFile(testFile, []byte(`{"level":"info","msg":"Test"}`), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	provider := NewFileProvider()
	defer provider.Close()

	if err := provider.Watch([]string{testFile}); err != nil {
		t.Fatalf("Failed to watch file: %v", err)
	}

	// Проверяем начальное состояние
	if !provider.IsSourceEnabled(testFile) {
		t.Error("Source should be enabled by default")
	}

	// Переключаем
	provider.ToggleSource(testFile)

	if provider.IsSourceEnabled(testFile) {
		t.Error("Source should be disabled after toggle")
	}

	// Переключаем обратно
	provider.ToggleSource(testFile)

	if !provider.IsSourceEnabled(testFile) {
		t.Error("Source should be enabled after second toggle")
	}
}

// BenchmarkExpandPaths бенчмарк для ExpandPaths.
func BenchmarkExpandPaths(b *testing.B) {
	tmpDir := b.TempDir()

	// Создаём 100 файлов
	for i := 0; i < 100; i++ {
		path := filepath.Join(tmpDir, "test"+string(rune(i))+".log")
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			b.Fatalf("Failed to create test file: %v", err)
		}
	}

	pattern := filepath.Join(tmpDir, "*.log")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExpandPaths([]string{pattern})
	}
}

// BenchmarkFileProviderThroughput бенчмарк производительности FileProvider.
func BenchmarkFileProviderThroughput(b *testing.B) {
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "benchmark.log")

	// Создаём большой файл
	content := ""
	for i := 0; i < b.N; i++ {
		content += `{"level":"info","msg":"Benchmark message"}
`
	}

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	provider := NewFileProvider()
	defer provider.Close()

	if err := provider.Watch([]string{testFile}); err != nil {
		b.Fatalf("Failed to watch file: %v", err)
	}

	b.ResetTimer()
	count := 0
	timeout := time.After(5 * time.Second)
	for {
		select {
		case _, ok := <-provider.LogChan():
			if !ok {
				return
			}
			count++
		case <-timeout:
			b.Logf("Processed %d lines in 5 seconds", count)
			return
		}
	}
}
