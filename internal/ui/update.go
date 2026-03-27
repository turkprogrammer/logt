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
		m.Provider.Close()
		return m, tea.Quit

	case tea.KeySpace:
		m.TogglePause()
		return m, ReadLogs(m.Provider)

	case tea.KeyEscape:
		if m.FilterMode != FilterNone {
			m.FilterMode = FilterNone
			m.FilterText = ""
			m.RegexPattern = nil
			m.RegexError = ""
			m.UpdateSearchMatches()
		}
		return m, nil

	case tea.KeyRunes:
		key := string(msg.Runes)

		runes := msg.Runes
		if len(runes) > 0 {
			r := runes[0]

			switch {
			case r == 'G' && m.FilterMode == FilterNone:
				m.GoToEnd()
				return m, nil
			case r == 'N' && m.FilterMode == FilterNone:
				m.NavigateToMatch(-1)
				return m, nil
			case r == 'n' && m.FilterMode == FilterNone:
				m.NavigateToMatch(1)
				return m, nil
			case r == 'g' && m.FilterMode == FilterNone:
				m.GoToStart()
				return m, nil
			}
		}

		switch key {
		case "/":
			if m.FilterMode == FilterNone {
				m.FilterMode = FilterInput
				m.FilterText = ""
				m.RegexPattern = nil
				m.RegexError = ""
			}
		case "r":
			if m.FilterMode == FilterNone {
				m.FilterMode = FilterRegex
				m.FilterText = ""
				m.RegexPattern = nil
				m.RegexError = ""
			} else if m.FilterMode == FilterInput {
				m.FilterMode = FilterRegex
				m.RegexPattern = nil
				m.RegexError = ""
			} else if m.FilterMode == FilterRegex {
				m.FilterMode = FilterInput
				m.RegexPattern = nil
				m.RegexError = ""
			}
		default:
			if m.FilterMode != FilterNone {
				m.FilterText += key
				m.UpdateSearchMatches()
			}
		}
		return m, nil

	case tea.KeyEnter:
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

	case tea.KeyBackspace:
		if m.FilterMode != FilterNone && len(m.FilterText) > 0 {
			m.FilterText = m.FilterText[:len(m.FilterText)-1]
			m.UpdateSearchMatches()
		}
		return m, nil

	case tea.KeyUp:
		if m.FilterMode != FilterNone {
			return m, nil
		}
		m.ScrollUp(1)
		return m, nil

	case tea.KeyDown:
		if m.FilterMode != FilterNone {
			return m, nil
		}
		m.ScrollDown(1)
		return m, nil

	case tea.KeyPgUp:
		m.ScrollUp(10)
		return m, nil

	case tea.KeyPgDown:
		m.ScrollDown(10)
		return m, nil

	case tea.KeyHome:
		m.GoToStart()
		return m, nil

	case tea.KeyEnd:
		m.GoToEnd()
		return m, nil

	case tea.KeyTab:
		m.ShowSourcePanel = !m.ShowSourcePanel
		return m, nil
	}

	return m, nil
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
