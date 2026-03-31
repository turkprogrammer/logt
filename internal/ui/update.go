// Package ui реализует пользовательский интерфейс на основе Bubble Tea TUI framework.
package ui

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/turkprogrammer/logt/internal/domain"
	"github.com/turkprogrammer/logt/internal/provider"
)

// MsgLogLine представляет сообщение о новой строке лога.
type MsgLogLine struct {
	Line domain.LogLine
}

// ReadLogs создаёт команду для чтения логов из провайдера.
func ReadLogs(p provider.Provider) tea.Cmd {
	return func() tea.Msg {
		for logLine := range p.LogChan() {
			return MsgLogLine{Line: logLine}
		}
		return nil
	}
}

// Update обрабатывает сообщения от Bubble Tea.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, ReadLogs(m.Provider)

	case tea.KeyMsg:
		return m.handleKey(msg)

	case MsgLogLine:
		if !m.Paused {
			m.Buffer.Add(msg.Line)
			if ShouldAutoScroll(m) {
				lines := m.VisibleLines()
				m.SelectedLine = len(lines) - 1
			}
			m.UpdateSearchMatches()
		}
		return m, ReadLogs(m.Provider)
	}

	return m, nil
}

// handleKey обрабатывает нажатия клавиш.
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.ExpandedJSON != nil {
		return m.handleJSONViewKey(msg)
	}

	switch msg.Type {
	case tea.KeyCtrlC:
		return m.handleCtrlC()
	case tea.KeySpace:
		return m.handleSpace()
	case tea.KeyEscape:
		return m.handleEscape()
	case tea.KeyEnter:
		return m.handleEnter()
	case tea.KeyBackspace:
		return m.handleBackspace()
	case tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown, tea.KeyHome, tea.KeyEnd:
		return m.handleNavigation(msg)
	case tea.KeyRunes:
		return m.handleRunes(msg)
	case tea.KeyTab:
		m.ShowSourcePanel = !m.ShowSourcePanel
		return m, nil
	}

	return m, nil
}

// handleCtrlC обрабатывает Ctrl+C (выход).
func (m *Model) handleCtrlC() (tea.Model, tea.Cmd) {
	m.Provider.Close()
	return m, tea.Quit
}

// handleSpace обрабатывает Space (пауза).
func (m *Model) handleSpace() (tea.Model, tea.Cmd) {
	m.TogglePause()
	return m, ReadLogs(m.Provider)
}

// handleEscape обрабатывает Escape (сброс фильтра).
func (m *Model) handleEscape() (tea.Model, tea.Cmd) {
	if m.BookmarkView {
		m.BookmarkView = false
		return m, nil
	}
	if m.FilterMode != FilterNone {
		m.FilterMode = FilterNone
		m.FilterText = ""
		m.RegexPattern = nil
		m.RegexError = ""
		m.UpdateSearchMatches()
	}
	return m, nil
}

// handleEnter обрабатывает Enter (применить фильтр или открыть JSON).
func (m *Model) handleEnter() (tea.Model, tea.Cmd) {
	if m.FilterMode != FilterNone && m.FilterText != "" {
		if m.FilterMode == FilterRegex {
			if err := m.SetRegex(m.FilterText); err != nil {
				m.RegexError = "Invalid regex: " + err.Error()
			}
		}
		m.UpdateSearchMatches()
		m.FilterMode = FilterNone
	} else {
		lines := m.VisibleLines()
		if m.SelectedLine >= 0 && m.SelectedLine < len(lines) {
			line := lines[m.SelectedLine]
			if line.IsJSON {
				m.ExpandJSON(m.SelectedLine)
			}
		}
	}
	return m, nil
}

// handleBackspace обрабатывает Backspace (удаление символа фильтра).
func (m *Model) handleBackspace() (tea.Model, tea.Cmd) {
	if m.FilterMode != FilterNone && len(m.FilterText) > 0 {
		m.FilterText = m.FilterText[:len(m.FilterText)-1]
		m.UpdateSearchMatches()
	}
	return m, nil
}

// handleNavigation обрабатывает клавиши навигации.
func (m *Model) handleNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.FilterMode != FilterNone {
		return m, nil
	}
	switch msg.Type {
	case tea.KeyUp:
		m.ScrollUp(1)
	case tea.KeyDown:
		m.ScrollDown(1)
	case tea.KeyPgUp:
		m.ScrollUp(10)
	case tea.KeyPgDown:
		m.ScrollDown(10)
	case tea.KeyHome:
		m.GoToStart()
	case tea.KeyEnd:
		m.GoToEnd()
	}
	return m, nil
}

// handleRunes обрабатывает символьные клавиши (/, r, g, G, n, N).
func (m *Model) handleRunes(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	runes := msg.Runes
	if len(runes) == 0 {
		return m, nil
	}
	r := runes[0]

	// Обработка заглавных букв (G, N, n, g)
	if m.handleCapitalLetters(r) {
		return m, nil
	}

	key := string(runes)
	switch key {
	case "/":
		m.openFilterInput()
	case "r":
		m.toggleFilterMode()
	case "m":
		m.addBookmark()
	case "M":
		m.toggleBookmarkView()
	case "e":
		m.exportBookmarks()
	default:
		if m.FilterMode != FilterNone {
			m.FilterText += key
			m.UpdateSearchMatches()
		}
	}
	return m, nil
}

// handleCapitalLetters обрабатывает заглавные буквы (G, N, n, g).
func (m *Model) handleCapitalLetters(r rune) bool {
	switch r {
	case 'G':
		m.GoToEnd()
		return true
	case 'N':
		m.NavigateToMatch(-1)
		return true
	case 'n':
		m.NavigateToMatch(1)
		return true
	case 'g':
		m.GoToStart()
		return true
	}
	return false
}

// openFilterInput открывает ввод фильтра.
func (m *Model) openFilterInput() {
	if m.FilterMode == FilterNone {
		m.FilterMode = FilterInput
		m.FilterText = ""
		m.RegexPattern = nil
		m.RegexError = ""
	}
}

// addBookmark добавляет bookmark текущей строки.
func (m *Model) addBookmark() {
	if m.FilterMode == FilterNone && !m.BookmarkView {
		lines := m.VisibleLines()
		if m.SelectedLine >= 0 && m.SelectedLine < len(lines) {
			m.Bookmarks.Add(lines[m.SelectedLine], "")
		}
	}
}

// toggleBookmarkView переключает режим просмотра bookmarks.
func (m *Model) toggleBookmarkView() {
	if m.FilterMode == FilterNone {
		m.BookmarkView = !m.BookmarkView
	}
}

// exportBookmarks экспортирует bookmarks в файл.
func (m *Model) exportBookmarks() {
	if m.FilterMode == FilterNone && !m.BookmarkView {
		m.Bookmarks.Export("bookmarks.yaml")
	}
}

// toggleFilterMode переключает режимы фильтрации.
func (m *Model) toggleFilterMode() {
	switch m.FilterMode {
	case FilterNone:
		m.FilterMode = FilterRegex
		m.FilterText = ""
		m.RegexPattern = nil
		m.RegexError = ""
	case FilterInput:
		m.FilterMode = FilterRegex
		m.RegexPattern = nil
		m.RegexError = ""
	case FilterRegex:
		m.FilterMode = FilterInput
		m.RegexPattern = nil
		m.RegexError = ""
	}
}

// handleJSONViewKey обрабатывает навигацию в режиме просмотра JSON.
func (m *Model) handleJSONViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.CollapseJSON()
		return m, nil

	case tea.KeyRunes:
		if string(msg.Runes) == "q" {
			m.CollapseJSON()
			return m, nil
		}

	case tea.KeyUp:
		if m.ExpandedJSON.Selected > 0 {
			m.ExpandedJSON.Selected--
		}
		return m, nil

	case tea.KeyDown:
		if m.ExpandedJSON.Selected < len(m.ExpandedJSON.Keys)-1 {
			m.ExpandedJSON.Selected++
		}
		return m, nil

	case tea.KeyHome:
		m.ExpandedJSON.Selected = 0
		return m, nil

	case tea.KeyEnd:
		m.ExpandedJSON.Selected = len(m.ExpandedJSON.Keys) - 1
		return m, nil
	}

	return m, nil
}

// FuzzyMatch проксирует вызов в domain пакет.
var FuzzyMatch = domain.FuzzyMatch

// HighlightMatches проксирует вызов в domain пакет.
var HighlightMatches = domain.HighlightMatches
