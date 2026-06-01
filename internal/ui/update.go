package ui

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/turkprogrammer/logt/internal/domain"
	"github.com/turkprogrammer/logt/internal/provider"
)

// msgLogLine представляет сообщение о новой строке лога.
type msgLogLine struct {
	Line domain.LogLine
}

// readLogs создаёт команду для чтения логов из провайдера.
func readLogs(p provider.Provider) tea.Cmd {
	return func() tea.Msg {
		for logLine := range p.LogChan() {
			return msgLogLine{Line: logLine}
		}
		return nil
	}
}

// Update обрабатывает сообщения от Bubble Tea.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, readLogs(m.Provider)

	case tea.KeyMsg:
		return m.handleKey(msg)

	case msgLogLine:
		if !m.Paused {
			m.Buffer.Add(msg.Line)
			m.RateCalculator.Update()
			if shouldAutoScroll(m) {
				lines := m.VisibleLines()
				m.SelectedLine = len(lines) - 1
			}
			m.UpdateSearchMatches()
		}
		return m, readLogs(m.Provider)
	}

	return m, nil
}

// handleKey обрабатывает нажатия клавиш.
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.expandedJSON != nil {
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
	return m, readLogs(m.Provider)
}

// handleEscape обрабатывает Escape (сброс фильтра).
func (m *Model) handleEscape() (tea.Model, tea.Cmd) {
	if m.BookmarkView {
		m.BookmarkView = false
		return m, nil
	}
	if m.FilterMode != filterNone {
		m.FilterMode = filterNone
		m.FilterText = ""
		m.RegexPattern = nil
		m.RegexError = ""
		m.UpdateSearchMatches()
	}
	return m, nil
}

// handleEnter обрабатывает Enter (применить фильтр или открыть JSON).
func (m *Model) handleEnter() (tea.Model, tea.Cmd) {
	if m.FilterMode != filterNone && m.FilterText != "" {
		if m.FilterMode == filterRegex {
			if err := m.SetRegex(m.FilterText); err != nil {
				m.RegexError = "Invalid regex: " + err.Error()
			}
		}
		m.UpdateSearchMatches()
		m.FilterMode = filterNone
		return m, nil
	}

	lines := m.VisibleLines()
	if m.SelectedLine < 0 || m.SelectedLine >= len(lines) {
		return m, nil
	}
	if lines[m.SelectedLine].IsJSON {
		m.ExpandJSON(m.SelectedLine)
	}
	return m, nil
}

// handleBackspace обрабатывает Backspace (удаление символа фильтра).
func (m *Model) handleBackspace() (tea.Model, tea.Cmd) {
	if m.FilterMode != filterNone && len(m.FilterText) > 0 {
		m.FilterText = m.FilterText[:len(m.FilterText)-1]
		m.UpdateSearchMatches()
	}
	return m, nil
}

// handleNavigation обрабатывает клавиши навигации.
func (m *Model) handleNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.FilterMode != filterNone {
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
		m.openfilterInput()
	case "r":
		m.toggleFilterMode()
	case "m":
		m.addBookmark()
	case "M":
		m.toggleBookmarkView()
	case "e":
		m.exportBookmarks()
	default:
		if m.FilterMode != filterNone {
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

// openfilterInput открывает ввод фильтра.
func (m *Model) openfilterInput() {
	if m.FilterMode == filterNone {
		m.FilterMode = filterInput
		m.FilterText = ""
		m.RegexPattern = nil
		m.RegexError = ""
	}
}

// addBookmark добавляет bookmark текущей строки.
func (m *Model) addBookmark() {
	if m.FilterMode == filterNone && !m.BookmarkView {
		lines := m.VisibleLines()
		if m.SelectedLine >= 0 && m.SelectedLine < len(lines) {
			m.Bookmarks.Add(lines[m.SelectedLine], "")
		}
	}
}

// toggleBookmarkView переключает режим просмотра bookmarks.
func (m *Model) toggleBookmarkView() {
	if m.FilterMode == filterNone {
		m.BookmarkView = !m.BookmarkView
	}
}

// exportBookmarks экспортирует bookmarks в файл.
func (m *Model) exportBookmarks() {
	if m.FilterMode == filterNone && !m.BookmarkView {
		m.Bookmarks.Export("bookmarks.yaml")
	}
}

// toggleFilterMode переключает режимы фильтрации.
func (m *Model) toggleFilterMode() {
	switch m.FilterMode {
	case filterNone:
		m.FilterMode = filterRegex
		m.FilterText = ""
		m.RegexPattern = nil
		m.RegexError = ""
	case filterInput:
		m.FilterMode = filterRegex
		m.RegexPattern = nil
		m.RegexError = ""
	case filterRegex:
		m.FilterMode = filterInput
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
		if m.expandedJSON.Selected > 0 {
			m.expandedJSON.Selected--
		}
		return m, nil

	case tea.KeyDown:
		if m.expandedJSON.Selected < len(m.expandedJSON.Keys)-1 {
			m.expandedJSON.Selected++
		}
		return m, nil

	case tea.KeyHome:
		m.expandedJSON.Selected = 0
		return m, nil

	case tea.KeyEnd:
		m.expandedJSON.Selected = len(m.expandedJSON.Keys) - 1
		return m, nil
	}

	return m, nil
}
