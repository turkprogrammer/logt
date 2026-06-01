// Package ui реализует пользовательский интерфейс на основе Bubble Tea TUI framework.
//
// Основные компоненты:
//   - Model: состояние приложения
//   - Update: обработка сообщений и событий
//   - View: рендеринг интерфейса с помощью Lip Gloss
package ui

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/turkprogrammer/logt/internal/domain"
	"github.com/turkprogrammer/logt/internal/domain/jsonpath"
	"github.com/turkprogrammer/logt/internal/provider"
)

// filterMode определяет режим работы фильтра.
type filterMode int

// Режимы фильтрации.
const (
	filterNone  filterMode = iota // Фильтр выключен
	filterInput                   // Ввод фильтра
	filterRegex                   // Режим регулярных выражений
)

// expandedJSON хранит состояние развёрнутого JSON просмотра.
type expandedJSON struct {
	Line     domain.LogLine // Исходная строка
	Selected int            // Выбранный ключ
	Keys     []string       // Ключи JSON
	Data     map[string]any // Данные JSON
}

// Model представляет состояние TUI приложения LogT.
type Model struct {
	Buffer   *domain.RingBuffer // Кольцевой буфер логов
	Provider provider.Provider  // Провайдер данных
	Width    int                // Ширина терминала
	Height   int                // Высота терминала

	Paused       bool           // Режим паузы
	AutoScroll   bool           // Автопрокрутка
	FilterMode   filterMode     // Режим фильтра
	FilterText   string         // Текст фильтра
	RegexPattern *regexp.Regexp // Скомпилированный regex
	RegexError   string         // Ошибка regex

	SelectedLine    int            // Выбранная строка
	expandedJSON    *expandedJSON  // Развёрнутый JSON
	ShowSourcePanel bool           // Показывать панель источников

	Sources        []domain.Source // Список источников
	IncludeSources map[string]bool // Какие источники показывать

	SearchMatches []int // Индексы совпадений
	CurrentMatch  int   // Текущее совпадение

	// Временные фильтры
	Since *time.Time
	Until *time.Time

	// JSON Path фильтр
	JSONFilter *jsonpath.Filter

	// Bookmarks
	Bookmarks    *domain.BookmarkManager
	BookmarkView bool // Режим просмотра bookmarks

	// Rate calculator
	RateCalculator *domain.RateCalculator
}

// NewModel создаёт новую модель с указанным провайдером.
func NewModel(p provider.Provider, since, until *time.Time, jsonFilter *jsonpath.Filter) *Model {
	sources := p.Sources()
	includeSources := make(map[string]bool)
	for _, s := range sources {
		includeSources[s.Path] = true
	}

	buf := p.Buffer()
	if buf == nil {
		buf = domain.NewRingBuffer(5000)
	}

	return &Model{
		Buffer:          buf,
		Provider:        p,
		Paused:          false,
		AutoScroll:      true,
		FilterMode:      filterNone,
		FilterText:      "",
		SelectedLine:    0,
		ShowSourcePanel: false,
		Sources:         sources,
		IncludeSources:  includeSources,
		SearchMatches:   []int{},
		CurrentMatch:    -1,
		Since:           since,
		Until:           until,
		JSONFilter:      jsonFilter,
		Bookmarks:       domain.NewBookmarkManager(""),
		BookmarkView:    false,
		RateCalculator:  domain.NewRateCalculator(),
	}
}

// SetSize устанавливает размеры терминала.
func (m *Model) SetSize(width, height int) {
	m.Width = width
	m.Height = height
}

// VisibleLines возвращает видимые (отфильтрованные) строки.
func (m *Model) VisibleLines() []domain.LogLine {
	return m.Buffer.GetFilteredCombined(domain.FilterOptions{
		Text:           m.FilterText,
		IncludeSources: m.IncludeSources,
		Since:          m.Since,
		Until:          m.Until,
		JSONFilter:     m.JSONFilter,
	})
}

// VisibleBookmarkLines возвращает строки из bookmarks.
func (m *Model) VisibleBookmarkLines() []domain.LogLine {
	bookmarks := m.Bookmarks.GetAll()
	lines := make([]domain.LogLine, len(bookmarks))
	for i, b := range bookmarks {
		lines[i] = b.Line
	}
	return lines
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
	if m.FilterMode == filterRegex {
		m.FilterMode = filterNone
		m.RegexPattern = nil
		m.RegexError = ""
	} else {
		m.FilterMode = filterRegex
	}
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

	data, ok := line.Parsed.(map[string]any)
	if !ok {
		return
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	m.expandedJSON = &expandedJSON{
		Line:     line,
		Selected: 0,
		Keys:     keys,
		Data:     maps.Clone(data),
	}
}

// CollapseJSON закрывает просмотр JSON.
func (m *Model) CollapseJSON() {
	m.expandedJSON = nil
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
	lines := m.VisibleLines()
	matches := make([]int, 0, len(lines))
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

	// Rate информация
	rateInfo := ""
	if m.RateCalculator != nil {
		rate := m.RateCalculator.Rate()
		if rate > 0 {
			rateInfo = fmt.Sprintf(" | Rate: ~%.0f l/s", rate)
		}
	}

	return fmt.Sprintf("%sFiles: %d/%d | Lines: %d/%d%s%s",
		pausedInfo, enabledSources, len(m.Sources), len(lines), totalLines, filterInfo, rateInfo)
}

// Стили отображения для разных уровней логирования.
var (
	infoStyle     = lipgloss.NewStyle().Foreground(colorBlue)
	warnStyle     = lipgloss.NewStyle().Foreground(colorYellow)
	errorStyle    = lipgloss.NewStyle().Foreground(colorRed)
	debugStyle    = lipgloss.NewStyle().Foreground(colorSubtext)
	fatalStyle    = lipgloss.NewStyle().Foreground(colorRed).Bold(true)
	jsonStyle     = lipgloss.NewStyle().Foreground(colorMauve)
	sourceStyle   = lipgloss.NewStyle().Foreground(colorTeal)
	selectedStyle = lipgloss.NewStyle().Background(colorOverlay).Foreground(colorText)
)

// getLevelStyle возвращает стиль для указанного уровня логирования.
func getLevelStyle(level domain.LogLevel) lipgloss.Style {
	switch level {
	case domain.LevelInfo:
		return infoStyle
	case domain.LevelWarn:
		return warnStyle
	case domain.LevelError:
		return errorStyle
	case domain.LevelDebug:
		return debugStyle
	case domain.LevelFatal:
		return fatalStyle
	default:
		return lipgloss.NewStyle()
	}
}

// shouldAutoScroll проверяет, нужно ли автоматически прокручивать.
func shouldAutoScroll(m *Model) bool {
	return m.AutoScroll && !m.Paused && m.FilterMode == filterNone && m.expandedJSON == nil
}

// Init инициализирует модель для Bubble Tea.
func (m *Model) Init() tea.Cmd {
	return readLogs(m.Provider)
}
