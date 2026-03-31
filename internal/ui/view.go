// Package ui реализует пользовательский интерфейс на основе Bubble Tea TUI framework.
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/turkprogrammer/logt/internal/domain"
)

// Палитра цветов - приятная тёмная тема Catppuccin-inspired.
var (
	ColorBg      = lipgloss.Color("236") // #303446 - фон
	ColorSurface = lipgloss.Color("235") // #292c3e - панели
	ColorOverlay = lipgloss.Color("240") // #414559 - границы
	ColorText    = lipgloss.Color("225") // #c6d0f5 - основной текст
	ColorSubtext = lipgloss.Color("250") // #a6adc8 - вторичный текст
	ColorBlue    = lipgloss.Color("12")  // #89b4fa - синий (info)
	ColorGreen   = lipgloss.Color("10")  // #a6e3a1 - зелёный
	ColorYellow  = lipgloss.Color("11")  // #f9e2af - жёлтый (warn)
	ColorPeach   = lipgloss.Color("13")  // #fab387 - персиковый
	ColorRed     = lipgloss.Color("9")   // #f38ba8 - красный (error)
	ColorMauve   = lipgloss.Color("13")  // #cba6f7 - фиолетовый (JSON)
	ColorPink    = lipgloss.Color("5")   // #f5c2e7 - розовый
	ColorTeal    = lipgloss.Color("14")  // #94e2d5 - бирюзовый
)

// Стили отображения.
var (
	BorderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorOverlay).
			Background(ColorSurface)
	TitleStyle     = lipgloss.NewStyle().Bold(true).Foreground(ColorText)
	ContentStyle   = lipgloss.NewStyle().Foreground(ColorText)
	StatusBarStyle = lipgloss.NewStyle().
			Background(ColorOverlay).
			Foreground(ColorText).
			Padding(0, 1)
	FilterBarStyle = lipgloss.NewStyle().
			Background(ColorBlue).
			Foreground(ColorBg).
			Padding(0, 1)
	RegexBarStyle = lipgloss.NewStyle().
			Background(ColorMauve).
			Foreground(ColorBg).
			Padding(0, 1)
	JSONViewStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.DoubleBorder()).
			BorderForeground(ColorMauve).
			Background(ColorSurface).
			Foreground(ColorText).
			Padding(1, 2)
	JSONKeyStyle     = lipgloss.NewStyle().Foreground(ColorBlue).Bold(false)
	JSONValueStr     = lipgloss.NewStyle().Foreground(ColorGreen)
	JSONValueNum     = lipgloss.NewStyle().Foreground(ColorPeach)
	JSONValueBool    = lipgloss.NewStyle().Foreground(ColorYellow)
	SourcePanelStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(ColorTeal).
				Background(ColorSurface).
				Padding(1).
				Width(30)
	EmptyStyle = lipgloss.NewStyle().
			Foreground(ColorSubtext).
			AlignVertical(lipgloss.Center).
			AlignHorizontal(lipgloss.Center)
)

// View рендерит интерфейс приложения.
func (m *Model) View() string {
	if m.ExpandedJSON != nil {
		return m.renderJSONView()
	}

	// Bookmark view режим
	if m.BookmarkView {
		return m.renderBookmarkView()
	}

	var sb strings.Builder

	if m.FilterMode != FilterNone {
		sb.WriteString(m.renderFilterBar())
	}

	contentHeight := m.Height - 3

	if m.ShowSourcePanel {
		sourceWidth := 30
		logWidth := m.Width - sourceWidth - 1

		sourcePanel := m.renderSourcePanel()
		logView := m.renderLogView(logWidth, contentHeight)

		sb.WriteString(lipgloss.JoinHorizontal(
			lipgloss.Top,
			sourcePanel,
			lipgloss.NewStyle().Width(logWidth).Render(logView),
		))
	} else {
		sb.WriteString(m.renderLogView(m.Width, contentHeight))
	}

	sb.WriteString("\n")
	sb.WriteString(m.renderStatusBar())

	return sb.String()
}

// renderLogView рендерит список логов.
func (m *Model) renderLogView(width, height int) string {
	lines := m.VisibleLines()
	if len(lines) == 0 {
		emptyMsg := EmptyStyle.Width(width).Height(height).Render("No logs to display")
		return emptyMsg
	}

	start, end := m.calculateViewport(len(lines), height)

	var sb strings.Builder
	for i := start; i < end; i++ {
		lineStr := m.renderLogLine(lines[i], i, width)
		sb.WriteString(lineStr)
		if i < end-1 {
			sb.WriteString("\n")
		}
	}

	// Добавляем пустые строки для заполнения экрана
	for i := 0; i < height-(end-start); i++ {
		sb.WriteString("\n")
	}

	return sb.String()
}

// calculateViewport вычисляет диапазон видимых строк.
func (m *Model) calculateViewport(totalLines, height int) (start, end int) {
	start = 0
	end = totalLines

	if m.SelectedLine >= height {
		start = m.SelectedLine - height + 1
	}

	if end-start > height {
		end = start + height
	}

	if start < 0 {
		start = 0
	}
	if end > totalLines {
		end = totalLines
	}

	return start, end
}

// renderLogLine рендерит одну строку лога.
func (m *Model) renderLogLine(line domain.LogLine, index, width int) string {
	style := GetLevelStyle(line.Level)
	if line.IsJSON {
		style = lipgloss.NewStyle().Foreground(ColorMauve)
	}

	if index == m.SelectedLine {
		style = style.Background(ColorOverlay).Foreground(ColorText)
	}

	content := m.formatLine(line, width-5)

	lineNum := fmt.Sprintf("%4d ", index+1)
	sourceName := fmt.Sprintf(" %-12s", truncate(line.Source.Name, 12))

	lineStr := fmt.Sprintf("%s%s%s %s",
		lipgloss.NewStyle().Foreground(ColorOverlay).Render(lineNum),
		lipgloss.NewStyle().Foreground(ColorTeal).Render(sourceName),
		style.Render(truncate(content, width-20)),
		style.Render(getLevelTag(line.Level)),
	)

	return truncateWithANSI(lineStr, width)
}

// formatLine форматирует строку для отображения.
func (m *Model) formatLine(line domain.LogLine, maxWidth int) string {
	content := line.Content

	if line.IsJSON {
		if data, ok := line.Parsed.(map[string]interface{}); ok {
			if msg, ok := data["message"].(string); ok {
				content = msg
			} else if msg, ok := data["msg"].(string); ok {
				content = msg
			} else {
				content = formatJSONCompact(data)
			}
		}
	}

	return truncate(content, maxWidth)
}

// formatJSONCompact форматирует JSON компактно.
func formatJSONCompact(data map[string]interface{}) string {
	parts := make([]string, 0, len(data))
	for k, v := range data {
		switch val := v.(type) {
		case string:
			parts = append(parts, fmt.Sprintf("%s=%q", k, truncate(val, 50)))
		case float64:
			parts = append(parts, fmt.Sprintf("%s=%.2f", k, val))
		default:
			parts = append(parts, fmt.Sprintf("%s=%v", k, truncate(fmt.Sprintf("%v", val), 50)))
		}
	}
	return strings.Join(parts, " ")
}

// renderFilterBar рендерит строку ввода фильтра.
func (m *Model) renderFilterBar() string {
	var barStyle lipgloss.Style
	prompt := "/"

	if m.FilterMode == FilterRegex {
		barStyle = RegexBarStyle
		prompt = ".* "
	} else {
		barStyle = FilterBarStyle
		prompt = "/ "
	}

	cursor := " "
	if m.FilterMode == FilterInput || m.FilterMode == FilterRegex {
		cursor = "\u2588"
	}

	input := fmt.Sprintf("%s%s%s", barStyle.Render(prompt), m.FilterText, cursor)

	if m.RegexError != "" {
		input += " " + lipgloss.NewStyle().Foreground(ColorRed).Render(m.RegexError)
	}

	return input + "\n"
}

// renderStatusBar рендерит статус-бар.
func (m *Model) renderStatusBar() string {
	status := m.StatusText()
	return StatusBarStyle.Width(m.Width).Render(status)
}

// renderSourcePanel рендерит панель источников.
func (m *Model) renderSourcePanel() string {
	var sb strings.Builder

	title := SourcePanelStyle.Render("Sources")
	sb.WriteString(title)
	sb.WriteString("\n")

	for _, source := range m.Sources {
		enabled := m.IncludeSources[source.Path]
		checkbox := "[ ]"
		if enabled {
			checkbox = "[x]"
		}

		name := truncate(source.Name, 25)
		line := fmt.Sprintf("%s %s", checkbox, name)

		if !enabled {
			line = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(line)
		}

		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String()
}

// renderBookmarkView рендерит режим просмотра bookmarks.
func (m *Model) renderBookmarkView() string {
	lines := m.VisibleBookmarkLines()

	if len(lines) == 0 {
		emptyMsg := EmptyStyle.Width(m.Width).Height(m.Height).Render("No bookmarks\n\nPress 'm' to bookmark a line")
		return emptyMsg
	}

	var sb strings.Builder

	// Заголовок
	header := fmt.Sprintf(" Bookmarks (%d) ", len(lines))
	sb.WriteString(lipgloss.NewStyle().
		Background(ColorMauve).
		Foreground(ColorBg).
		Bold(true).
		Padding(0, 1).
		Render(header))
	sb.WriteString("\n")

	// Вывод bookmarks
	for i, line := range lines {
		style := GetLevelStyle(line.Level)
		if i == m.SelectedLine {
			style = style.Background(ColorOverlay).Foreground(ColorText)
		}

		lineNum := fmt.Sprintf("%4d ", i+1)
		content := truncate(line.Content, m.Width-20)

		lineStr := fmt.Sprintf("%s%s %s",
			lipgloss.NewStyle().Foreground(ColorOverlay).Render(lineNum),
			style.Render(content),
			style.Render(getLevelTag(line.Level)),
		)

		sb.WriteString(lineStr)
		sb.WriteString("\n")
	}

	// Подсказка
	sb.WriteString("\n")
	sb.WriteString(StatusBarStyle.Width(m.Width).Render("ESC: close | m: add | e: export | ↑↓: navigate"))

	return sb.String()
}

// renderJSONView рендерит развёрнутый JSON с подсветкой.
func (m *Model) renderJSONView() string {
	ej := m.ExpandedJSON
	if ej == nil {
		return ""
	}

	var sb strings.Builder

	// Заголовок
	sb.WriteString(m.renderJSONHeader(ej.Line.Source.Name))
	sb.WriteString("\n")

	// Разделитель
	separator := lipgloss.NewStyle().Foreground(ColorOverlay).Render(strings.Repeat("─", min(m.Width, 60)))
	sb.WriteString(separator)
	sb.WriteString("\n")

	// Ключи JSON
	for i, key := range ej.Keys {
		sb.WriteString(m.renderJSONKeyLine(key, ej.Data[key], i == ej.Selected))
	}

	// Нижний разделитель
	sb.WriteString(separator)
	sb.WriteString("\n")

	// Подсказка
	sb.WriteString(StatusBarStyle.Width(m.Width).Render("ESC/Q: close | ↑↓: navigate | Enter: copy value"))

	return sb.String()
}

// renderJSONHeader рендерит заголовок JSON view.
func (m *Model) renderJSONHeader(sourceName string) string {
	header := fmt.Sprintf(" JSON: %s ", sourceName)
	return lipgloss.NewStyle().
		Background(ColorMauve).
		Foreground(ColorBg).
		Bold(true).
		Padding(0, 1).
		Render(header)
}

// renderJSONKeyLine рендерит строку с ключом JSON.
func (m *Model) renderJSONKeyLine(key string, value interface{}, selected bool) string {
	prefix := "  "
	if selected {
		prefix = "▶ "
	}

	keyStr := JSONKeyStyle.Render(truncate(key, 20))
	valueStr := formatJSONValue(value)

	line := fmt.Sprintf("%s%s%s  %s",
		lipgloss.NewStyle().Foreground(ColorSubtext).Render(prefix),
		keyStr,
		lipgloss.NewStyle().Foreground(ColorOverlay).Render(":"),
		valueStr,
	)

	if selected {
		line += " ←"
	}

	return line + "\n"
}

// formatJSONValue форматирует значение JSON с подсветкой.
func formatJSONValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return JSONValueStr.Render(fmt.Sprintf("%q", truncate(val, 50)))
	case float64:
		return JSONValueNum.Render(fmt.Sprintf("%.0f", val))
	case bool:
		return JSONValueBool.Render(fmt.Sprintf("%t", val))
	case nil:
		return lipgloss.NewStyle().Foreground(ColorSubtext).Render("null")
	default:
		return lipgloss.NewStyle().Foreground(ColorSubtext).Render(fmt.Sprintf("%v", val))
	}
}

// getLevelTag возвращает тег уровня для отображения.
func getLevelTag(level domain.LogLevel) string {
	switch level {
	case domain.LevelDebug:
		return " DBG"
	case domain.LevelInfo:
		return " INF"
	case domain.LevelWarn:
		return " WRN"
	case domain.LevelError:
		return " ERR"
	case domain.LevelFatal:
		return " FTL"
	default:
		return ""
	}
}

// truncate обрезает строку до максимальной ширины.
func truncate(s string, maxLen int) string {
	visibleLen := runewidth.StringWidth(s)
	if visibleLen <= maxLen {
		return s
	}

	result := ""
	for _, r := range s {
		if runewidth.StringWidth(result)+runewidth.RuneWidth(r) > maxLen-2 {
			break
		}
		result += string(r)
	}
	return result + ".."
}

// truncateWithANSI обрезает строку с учётом ANSI кодов.
func truncateWithANSI(s string, maxLen int) string {
	visibleLen := runewidth.StringWidth(stripANSI(s))
	if visibleLen <= maxLen {
		return s
	}

	result := ""
	visibleCount := 0
	inANSI := false

	for _, r := range s {
		if r == '\x02' || r == '\x03' {
			result += string(r)
			continue
		}
		if r == '[' {
			inANSI = true
			result += string(r)
			continue
		}
		if inANSI {
			result += string(r)
			if r == 'm' {
				inANSI = false
			}
			continue
		}

		if visibleCount >= maxLen-2 {
			result += ".."
			break
		}

		result += string(r)
		visibleCount += runewidth.RuneWidth(r)
	}

	return result
}

// stripANSI удаляет ANSI коды из строки.
func stripANSI(s string) string {
	var result strings.Builder
	inANSI := false

	for _, r := range s {
		if r == '\x02' || r == '\x03' {
			continue
		}
		if r == '[' {
			inANSI = true
			continue
		}
		if inANSI {
			if r == 'm' {
				inANSI = false
			}
			continue
		}
		result.WriteRune(r)
	}

	return result.String()
}

// min возвращает минимум из двух чисел.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
