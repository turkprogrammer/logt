// Package domain тестирует менеджер bookmarks.
package domain

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBookmarkManager_Add(t *testing.T) {
	bm := NewBookmarkManager("")

	line := LogLine{
		Timestamp: time.Now(),
		Level:     LevelError,
		Content:   "Test error message",
		Source:    Source{Name: "test.log", Path: "/test/test.log"},
	}

	bm.Add(line, "Important error")

	bookmarks := bm.GetAll()
	if len(bookmarks) != 1 {
		t.Errorf("Expected 1 bookmark, got %d", len(bookmarks))
	}

	if bookmarks[0].Note != "Important error" {
		t.Errorf("Expected note 'Important error', got %q", bookmarks[0].Note)
	}
}

func TestBookmarkManager_GetAll(t *testing.T) {
	bm := NewBookmarkManager("")

	// Добавляем 3 bookmarks
	for i := 0; i < 3; i++ {
		bm.Add(LogLine{
			Content:   "Test message",
			Timestamp: time.Now(),
			Level:     LevelInfo,
		}, "Note")
	}

	bookmarks := bm.GetAll()
	if len(bookmarks) != 3 {
		t.Errorf("Expected 3 bookmarks, got %d", len(bookmarks))
	}
}

func TestBookmarkManager_Export(t *testing.T) {
	bm := NewBookmarkManager("")

	line := LogLine{
		Timestamp: time.Now(),
		Level:     LevelError,
		Content:   "Export test",
		Source:    Source{Name: "test.log"},
	}

	bm.Add(line, "Export note")

	// Создаём временный файл
	tmpDir := t.TempDir()
	exportPath := filepath.Join(tmpDir, "bookmarks.yaml")

	err := bm.Export(exportPath)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Проверяем, что файл существует
	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		t.Error("Export file does not exist")
	}

	// Читаем и проверяем содержимое
	content, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Export file is empty")
	}
}

func TestBookmarkManager_Load(t *testing.T) {
	bm := NewBookmarkManager("")

	// Создаём временный файл с тестовыми данными
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_bookmarks.yaml")

	testContent := `bookmarks:
  - line:
      content: "Test loaded message"
      level: "ERROR"
      timestamp: "2024-01-15T10:00:00Z"
    note: "Test note"
    created_at: "2024-01-15T10:00:00Z"
`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Загружаем
	err = bm.Load(testFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	bookmarks := bm.GetAll()
	if len(bookmarks) != 1 {
		t.Errorf("Expected 1 bookmark after load, got %d", len(bookmarks))
	}
}

func TestBookmarkManager_Load_EmptyFile(t *testing.T) {
	bm := NewBookmarkManager("")

	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty.yaml")

	err := os.WriteFile(emptyFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	// Загрузка пустого файла не должна вызывать ошибку
	err = bm.Load(emptyFile)
	if err != nil {
		t.Fatalf("Load empty file failed: %v", err)
	}

	bookmarks := bm.GetAll()
	if len(bookmarks) != 0 {
		t.Errorf("Expected 0 bookmarks from empty file, got %d", len(bookmarks))
	}
}

func TestBookmarkManager_Remove(t *testing.T) {
	bm := NewBookmarkManager("")

	line := LogLine{
		Content:   "To be removed",
		Timestamp: time.Now(),
		Level:     LevelInfo,
	}

	bm.Add(line, "Note 1")
	bm.Add(line, "Note 2")

	if len(bm.GetAll()) != 2 {
		t.Errorf("Expected 2 bookmarks, got %d", len(bm.GetAll()))
	}

	bm.Remove(0)

	if len(bm.GetAll()) != 1 {
		t.Errorf("Expected 1 bookmark after remove, got %d", len(bm.GetAll()))
	}
}

func TestBookmarkManager_Clear(t *testing.T) {
	bm := NewBookmarkManager("")

	// Добавляем несколько bookmarks
	for i := 0; i < 5; i++ {
		bm.Add(LogLine{Content: "Test"}, "Note")
	}

	bm.Clear()

	bookmarks := bm.GetAll()
	if len(bookmarks) != 0 {
		t.Errorf("Expected 0 bookmarks after clear, got %d", len(bookmarks))
	}
}
