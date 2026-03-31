// Package domain предоставляет менеджер bookmarks для сохранения важных строк логов.
package domain

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Bookmark представляет сохранённую строку лога с заметкой.
type Bookmark struct {
	Line      LogLine   // Исходная строка лога
	Note      string    // Заметка пользователя
	CreatedAt time.Time // Время создания bookmark
}

// BookmarkManager управляет коллекцией bookmarks.
type BookmarkManager struct {
	bookmarks []Bookmark
	path      string // Путь к файлу сохранения
}

// NewBookmarkManager создаёт новый BookmarkManager.
// Если path указан, bookmarks загружаются из файла.
func NewBookmarkManager(path string) *BookmarkManager {
	bm := &BookmarkManager{
		bookmarks: make([]Bookmark, 0),
		path:      path,
	}

	// Загружаем существующие bookmarks
	if path != "" {
		bm.Load(path)
	}

	return bm
}

// Add добавляет новую bookmark.
func (bm *BookmarkManager) Add(line LogLine, note string) {
	bm.bookmarks = append(bm.bookmarks, Bookmark{
		Line:      line,
		Note:      note,
		CreatedAt: time.Now(),
	})
}

// GetAll возвращает все bookmarks.
func (bm *BookmarkManager) GetAll() []Bookmark {
	result := make([]Bookmark, len(bm.bookmarks))
	copy(result, bm.bookmarks)
	return result
}

// Remove удаляет bookmark по индексу.
func (bm *BookmarkManager) Remove(index int) {
	if index < 0 || index >= len(bm.bookmarks) {
		return
	}
	bm.bookmarks = append(bm.bookmarks[:index], bm.bookmarks[index+1:]...)
}

// Clear очищает все bookmarks.
func (bm *BookmarkManager) Clear() {
	bm.bookmarks = make([]Bookmark, 0)
}

// Export экспортирует bookmarks в YAML файл.
func (bm *BookmarkManager) Export(path string) error {
	data, err := yaml.Marshal(bm.toYAML())
	if err != nil {
		return err
	}

	// Создаём директорию если не существует
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Load загружает bookmarks из YAML файла.
func (bm *BookmarkManager) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Файл не существует — это нормально
		}
		return err
	}

	if len(data) == 0 {
		return nil
	}

	var yamlData struct {
		Bookmarks []yamlBookmark `yaml:"bookmarks"`
	}

	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return err
	}

	bm.bookmarks = make([]Bookmark, 0, len(yamlData.Bookmarks))
	for _, yb := range yamlData.Bookmarks {
		bm.bookmarks = append(bm.bookmarks, yb.ToBookmark())
	}

	return nil
}

// yamlBookmark представляет bookmark для YAML сериализации.
type yamlBookmark struct {
	Line      yamlLogLine `yaml:"line"`
	Note      string      `yaml:"note"`
	CreatedAt time.Time   `yaml:"created_at"`
}

// yamlLogLine представляет LogLine для YAML сериализации.
type yamlLogLine struct {
	Content   string    `yaml:"content"`
	Level     string    `yaml:"level"`
	Timestamp time.Time `yaml:"timestamp"`
	Source    string    `yaml:"source"`
}

// toYAML конвертирует BookmarkManager в YAML структуру.
func (bm *BookmarkManager) toYAML() map[string][]yamlBookmark {
	yamlBookmarks := make([]yamlBookmark, len(bm.bookmarks))
	for i, b := range bm.bookmarks {
		yamlBookmarks[i] = yamlBookmark{
			Line: yamlLogLine{
				Content:   b.Line.Content,
				Level:     string(b.Line.Level),
				Timestamp: b.Line.Timestamp,
				Source:    b.Line.Source.Name,
			},
			Note:      b.Note,
			CreatedAt: b.CreatedAt,
		}
	}
	return map[string][]yamlBookmark{
		"bookmarks": yamlBookmarks,
	}
}

// ToBookmark конвертирует yamlBookmark в Bookmark.
func (yb yamlBookmark) ToBookmark() Bookmark {
	return Bookmark{
		Line: LogLine{
			Content:   yb.Line.Content,
			Level:     LogLevel(yb.Line.Level),
			Timestamp: yb.Line.Timestamp,
			Source:    Source{Name: yb.Line.Source},
		},
		Note:      yb.Note,
		CreatedAt: yb.CreatedAt,
	}
}
