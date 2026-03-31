// Package ui реализует пользовательский интерфейс на основе Bubble Tea TUI framework.
//
// Основные компоненты:
//   - Model: состояние приложения
//   - Update: обработка сообщений и событий
//   - View: рендеринг интерфейса с помощью Lip Gloss
package ui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/turkprogrammer/logt/internal/domain"
	"github.com/turkprogrammer/logt/internal/provider"
)

// FilterMode определяет режим работы фильтра.
type FilterMode int

// Режимы фильтрации.
const (
	FilterNone  FilterMode = iota // Фильтр выключен
	FilterInput                   // Ввод фильтра
	FilterRegex                   // Режим регулярных выражений
)

// ExpandedJSON хранит состояние развёрнутого JSON просмотра.
type ExpandedJSON struct {
	Line     domain.LogLine         // Исходная строка
	Selected int                    // Выбранный ключ
	Keys     []string               // Ключи JSON
	Data     map[string]interface{} // Данные JSON
}

// Model представляет состояние TUI приложения LogT.
type Model struct {
	Buffer   *domain.RingBuffer // Кольцевой буфер логов
	Provider provider.Provider  // Провайдер данных
	Width    int                // Ширина терминала
	Height   int                // Высота терминала

	Paused       bool           // Режим паузы
	AutoScroll   bool           // Автопрокрутка
	FilterMode   FilterMode     // Режим фильтра
	FilterText   string         // Текст фильтра
	RegexPattern *regexp.Regexp // Скомпилированный regex
	RegexError   string         // Ошибка regex

	SelectedLine    int           // Выбранная строка
	ExpandedJSON    *ExpandedJSON // Развёрнутый JSON
	ShowSourcePanel bool          // Показывать панель источников

	Sources        []domain.Source // Список источников
	IncludeSources map[string]bool // Какие источники показывать

	ThrottleDuration time.Duration // Минимум времени между обновлениями
	lastUpdate       time.Time     // Время последнего обновления

	SearchMatches []int  // Индексы совпадений
	CurrentMatch  int    // Текущее совпадение
	SearchPattern string // Паттерн поиска

	// Временные фильтры
	Since *time.Time
	Until *time.Time
}

// NewModel создаёт новую модель с указанным провайдером.
func NewModel(p provider.Provider, since, until *time.Time) *Model {
	sources := p.Sources()
	includeSources := make(map[string]bool)
	for _, s := range sources {
		includeSources[s.Path] = true
	}

	return &Model{
		Buffer:           domain.NewRingBuffer(5000),
		Provider:         p,
		Paused:           false,
		AutoScroll:       true,
		FilterMode:       FilterNone,
		FilterText:       "",
		SelectedLine:     0,
		ShowSourcePanel:  false,
		Sources:          sources,
		IncludeSources:   includeSources,
		ThrottleDuration: 50 * time.Millisecond,
		lastUpdate:       time.Now(),
		SearchMatches:    make([]int, 0),
		CurrentMatch:     -1,
		Since:            since,
		Until:            until,
	}
}

// SetSize устанавливает размеры терминала.
func (m *Model) SetSize(width, height int) {
	m.Width = width
	m.Height = height
}

// VisibleLines возвращает видимые (отфильтрованные) строки.
func (m *Model) VisibleLines() []domain.LogLine {
	return m.Buffer.GetFilteredWithTime(m.FilterText, m.IncludeSources, m.Since, m.Until)
}

// TogglePause переключает режим паузы.
func (m *Model) TogglePause() {
	m.Paused = !m.Paused
}

// ToggleSource переключает источник.
func (m *Model) ToggleSource(path string) {
	m.Provider.ToggleSource(path)
	m.IncludeSources = m.Provider.EnabledSources()
}

// SetFilter устанавливает фильтр.
func (m *Model) SetFilter(filter string) {
	m.FilterText = filter
	m.SelectedLine = 0
}

// SetRegex компилирует и устанавливает regex паттерн.
func (m *Model) SetRegex(pattern string) error {
	if pattern == "" {
		m.RegexPattern = nil
		m.RegexError = ""
		m.FilterText = ""
		return nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		m.RegexError = fmt.Sprintf("Invalid regex: %v", err)
		return err
	}

	m.RegexPattern = re
	m.FilterText = pattern
	m.RegexError = ""
	m.SelectedLine = 0
	return nil
}

// ToggleRegexMode переключает режим regex.
func (m *Model) ToggleRegexMode() {
	if m.FilterMode == FilterRegex {
		m.FilterMode = FilterNone
		m.RegexPattern = nil
		m.RegexError = ""
	} else {
		m.FilterMode = FilterRegex
	}
}

// ShouldThrottle проверяет, нужно ли ограничивать частоту обновлений.
func (m *Model) ShouldThrottle() bool {
	now := time.Now()
	if now.Sub(m.lastUpdate) >= m.ThrottleDuration {
		m.lastUpdate = now
		return true
	}
	return false
}

// ExpandJSON разворачивает JSON для просмотра.
func (m *Model) ExpandJSON(lineIdx int) {
	lines := m.VisibleLines()
	if lineIdx < 0 || lineIdx >= len(lines) {
		return
	}

	line := lines[lineIdx]
	if !line.IsJSON {
		return
	}

	data, ok := line.Parsed.(map[string]interface{})
	if !ok {
		return
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sortKeys(keys)

	m.ExpandedJSON = &ExpandedJSON{
		Line:     line,
		Selected: 0,
		Keys:     keys,
		Data:     data,
	}
}

// CollapseJSON закрывает просмотр JSON.
func (m *Model) CollapseJSON() {
	m.ExpandedJSON = nil
}

// NavigateToMatch переходит к следующему/предыдущему совпадению.
func (m *Model) NavigateToMatch(direction int) {
	matches := m.SearchMatches
	if len(matches) == 0 {
		return
	}

	m.CurrentMatch += direction
	if m.CurrentMatch < 0 {
		m.CurrentMatch = len(matches) - 1
	} else if m.CurrentMatch >= len(matches) {
		m.CurrentMatch = 0
	}

	m.SelectedLine = matches[m.CurrentMatch]
}

// UpdateSearchMatches обновляет список совпадений.
func (m *Model) UpdateSearchMatches() {
	matches := make([]int, 0)
	lines := m.VisibleLines()
	pattern := strings.ToLower(m.FilterText)

	for i, line := range lines {
		if strings.Contains(strings.ToLower(line.Content), pattern) {
			matches = append(matches, i)
		}
	}

	m.SearchMatches = matches
	m.CurrentMatch = -1
}

// GoToStart переходит к началу списка.
func (m *Model) GoToStart() {
	m.SelectedLine = 0
}

// GoToEnd переходит к концу списка.
func (m *Model) GoToEnd() {
	lines := m.VisibleLines()
	if len(lines) > 0 {
		m.SelectedLine = len(lines) - 1
	}
}

// ScrollUp прокручивает вверх.
func (m *Model) ScrollUp(n int) {
	m.SelectedLine -= n
	if m.SelectedLine < 0 {
		m.SelectedLine = 0
	}
}

// ScrollDown прокручивает вниз.
func (m *Model) ScrollDown(n int) {
	lines := m.VisibleLines()
	m.SelectedLine += n
	if m.SelectedLine >= len(lines) {
		m.SelectedLine = len(lines) - 1
	}
}

// StatusText возвращает текст статус-бара.
func (m *Model) StatusText() string {
	lines := m.VisibleLines()
	totalLines := m.Buffer.Len()
	enabledSources := 0
	for _, v := range m.IncludeSources {
		if v {
			enabledSources++
		}
	}

	filterInfo := ""
	if m.FilterText != "" {
		filterInfo = fmt.Sprintf(" | Filter: %q", m.FilterText)
	}
	if m.RegexError != "" {
		filterInfo = fmt.Sprintf(" | Regex Error: %s", m.RegexError)
	}

	pausedInfo := ""
	if m.Paused {
		pausedInfo = "[PAUSED] "
	}

	return fmt.Sprintf("%sFiles: %d/%d | Lines: %d/%d%s",
		pausedInfo, enabledSources, len(m.Sources), len(lines), totalLines, filterInfo)
}

// sortKeys сортирует ключи для отображения.
func sortKeys(keys []string) {
	for i := 0; i < len(keys)-1; i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
}

// Стили отображения для разных уровней логирования.
var (
	InfoStyle     = lipgloss.NewStyle().Foreground(ColorBlue)
	WarnStyle     = lipgloss.NewStyle().Foreground(ColorYellow)
	ErrorStyle    = lipgloss.NewStyle().Foreground(ColorRed)
	DebugStyle    = lipgloss.NewStyle().Foreground(ColorSubtext)
	FatalStyle    = lipgloss.NewStyle().Foreground(ColorRed).Bold(true)
	JSONStyle     = lipgloss.NewStyle().Foreground(ColorMauve)
	SourceStyle   = lipgloss.NewStyle().Foreground(ColorTeal)
	SelectedStyle = lipgloss.NewStyle().Background(ColorOverlay).Foreground(ColorText)
)

// GetLevelStyle возвращает стиль для указанного уровня логирования.
func GetLevelStyle(level domain.LogLevel) lipgloss.Style {
	switch level {
	case domain.LevelInfo:
		return InfoStyle
	case domain.LevelWarn:
		return WarnStyle
	case domain.LevelError:
		return ErrorStyle
	case domain.LevelDebug:
		return DebugStyle
	case domain.LevelFatal:
		return FatalStyle
	default:
		return lipgloss.NewStyle()
	}
}

// ShouldAutoScroll проверяет, нужно ли автоматически прокручивать.
func ShouldAutoScroll(m *Model) bool {
	return m.AutoScroll && !m.Paused && m.FilterMode == FilterNone && m.ExpandedJSON == nil
}

// Init инициализирует модель для Bubble Tea.
func (m *Model) Init() tea.Cmd {
	return ReadLogs(m.Provider)
}
