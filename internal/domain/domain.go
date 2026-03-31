// Package domain содержит основные модели данных, парсеры и структуры для работы с логами.
//
// Основные компоненты:
//   - LogLine: структура одной строки лога
//   - RingBuffer: потокобезопасный кольцевой буфер для хранения логов
//   - Парсеры: JSON, Logfmt, Plain для различных форматов логов
package domain

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// LogLevel представляет уровень важности сообщения в логе.
type LogLevel string

// Константы уровней логирования.
const (
	LevelDebug   LogLevel = "DEBUG"   // Отладочные сообщения
	LevelInfo    LogLevel = "INFO"    // Информационные сообщения
	LevelWarn    LogLevel = "WARN"    // Предупреждения
	LevelError   LogLevel = "ERROR"   // Ошибки
	LevelFatal   LogLevel = "FATAL"   // Критические ошибки
	LevelTrace   LogLevel = "TRACE"   // Трассировка
	LevelUnknown LogLevel = "UNKNOWN" // Неизвестный уровень
)

// Source представляет источник логов (имя файла или stdin).
type Source struct {
	Name string // Имя источника (например, имя файла)
	Path string // Полный путь к источнику
}

// LogLine представляет одну строку лога с распарсенными данными.
type LogLine struct {
	Timestamp time.Time   // Время из лога
	Level     LogLevel    // Уровень логирования
	Source    Source      // Источник лога
	Content   string      // Текстовое содержимое строки
	Raw       string      // Исходная сырая строка
	Parsed    interface{} // Распарсенные данные (для JSON)
	IsJSON    bool        // Флаг, что строка является JSON
}

// Parser определяет интерфейс для парсеров логов.
type Parser interface {
	Parse(line string, source Source) *LogLine // Парсит строку в LogLine
	CanParse(line string) bool                 // Проверяет, может ли парсер обработать строку
}

// Паттерны для определения уровня логирования.
var levelPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\b(FATAL|CRITICAL)\b`),
	regexp.MustCompile(`\b(ERROR|ERR)\b`),
	regexp.MustCompile(`\b(WARN|WARNING)\b`),
	regexp.MustCompile(`\b(INFO)\b`),
	regexp.MustCompile(`\b(DEBUG|DBG)\b`),
	regexp.MustCompile(`\b(TRACE|VERBOSE)\b`),
}

// Паттерны для определения временной метки в строке лога.
var timestampPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}`),
	regexp.MustCompile(`^\[\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}`),
	regexp.MustCompile(`^\d{2}/\w{3}/\d{4}:\d{2}:\d{2}:\d{2}`),
}

// DetectLevel определяет уровень логирования по тексту строки.
// Возвращает LevelUnknown если уровень не определён.
// Поиск регистронезависимый.
func DetectLevel(line string) LogLevel {
	upper := strings.ToUpper(line)
	for _, pattern := range levelPatterns {
		if pattern.MatchString(upper) {
			switch {
			case strings.Contains(upper, "FATAL") || strings.Contains(upper, "CRITICAL"):
				return LevelFatal
			case strings.Contains(upper, "ERROR") || strings.Contains(upper, "ERR"):
				return LevelError
			case strings.Contains(upper, "WARN"):
				return LevelWarn
			case strings.Contains(upper, "INFO"):
				return LevelInfo
			case strings.Contains(upper, "DEBUG") || strings.Contains(upper, "DBG"):
				return LevelDebug
			case strings.Contains(upper, "TRACE") || strings.Contains(upper, "VERBOSE"):
				return LevelTrace
			}
		}
	}
	return LevelUnknown
}

// ParseTimestamp извлекает временную метку из строки лога.
// Поддерживает форматы: ISO 8601, Apache, и другие.
// Возвращает текущее время если парсинг не удался.
func ParseTimestamp(line string) time.Time {
	for _, pattern := range timestampPatterns {
		if match := pattern.FindString(line); match != "" {
			t, err := parseTimestampValue(match)
			if err == nil {
				return t
			}
		}
	}
	return time.Now()
}

// parseTimestampValue внутренняя функция для парсинга временной метки.
func parseTimestampValue(s string) (time.Time, error) {
	s = strings.TrimPrefix(s, "[")
	formats := []string{
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006/01/02:15:04:05",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse timestamp")
}

// IsValidJSON проверяет, является ли строка валидным JSON.
// Возвращает true если строка начинается с { или [ и валидна.
func IsValidJSON(line string) bool {
	line = strings.TrimSpace(line)
	if len(line) < 2 || (line[0] != '{' && line[0] != '[') {
		return false
	}
	var js json.RawMessage
	return json.Unmarshal([]byte(line), &js) == nil
}

// MultiParser объединяет несколько парсеров и выбирает подходящий.
type MultiParser struct {
	parsers []Parser // Список парсеров в порядке приоритета
}

// NewMultiParser создаёт новый MultiParser с предустановленными парсерами.
func NewMultiParser() *MultiParser {
	return &MultiParser{
		parsers: []Parser{
			&JSONParser{},
			&LogfmtParser{},
			&PlainParser{},
		},
	}
}

// Parse парсит строку используя первый подходящий парсер.
func (mp *MultiParser) Parse(line string, source Source) *LogLine {
	for _, p := range mp.parsers {
		if p.CanParse(line) {
			return p.Parse(line, source)
		}
	}
	return PlainParser{}.Parse(line, source)
}

// JSONParser парсит JSON-форматированные логи.
type JSONParser struct{}

// Parse парсит JSON строку в LogLine.
func (p *JSONParser) Parse(line string, source Source) *LogLine {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		return nil
	}

	logLine := &LogLine{
		Source:    source,
		Raw:       line,
		Content:   line,
		IsJSON:    true,
		Parsed:    data,
		Timestamp: ParseTimestamp(line),
		Level:     LevelUnknown,
	}

	// Извлекаем уровень из JSON полей
	if level, ok := data["level"].(string); ok {
		logLine.Level = LogLevel(strings.ToUpper(level))
	} else if level, ok := data["severity"].(string); ok {
		logLine.Level = LogLevel(strings.ToUpper(level))
	} else {
		logLine.Level = DetectLevel(line)
	}

	// Извлекаем временную метку из JSON полей
	if ts, ok := data["timestamp"].(string); ok {
		if t, err := parseTime(ts); err == nil {
			logLine.Timestamp = t
		}
	} else if ts, ok := data["time"].(string); ok {
		if t, err := parseTime(ts); err == nil {
			logLine.Timestamp = t
		}
	} else if ts, ok := data["@timestamp"].(string); ok {
		if t, err := parseTime(ts); err == nil {
			logLine.Timestamp = t
		}
	}

	return logLine
}

// CanParse проверяет, является ли строка валидным JSON.
func (p *JSONParser) CanParse(line string) bool {
	return IsValidJSON(line)
}

// parseTime парсит время в формате RFC3339.
func parseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// Паттерн для парсинга logfmt формата.
var logfmtPattern = regexp.MustCompile(`(\w+)=("[^"]*"|\S+)`)

// LogfmtParser парсит logfmt-форматированные логи.
type LogfmtParser struct{}

// Parse парсит logfmt строку в LogLine.
func (p *LogfmtParser) Parse(line string, source Source) *LogLine {
	matches := logfmtPattern.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return nil
	}

	data := make(map[string]string)
	for _, m := range matches {
		if len(m) == 3 {
			key := m[1]
			val := m[2]
			if strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`) {
				val = val[1 : len(val)-1]
			}
			data[key] = val
		}
	}

	logLine := &LogLine{
		Source:    source,
		Raw:       line,
		Content:   line,
		IsJSON:    false,
		Parsed:    data,
		Timestamp: ParseTimestamp(line),
		Level:     LevelUnknown,
	}

	if level, ok := data["level"]; ok {
		logLine.Level = LogLevel(strings.ToUpper(level))
	} else if level, ok := data["severity"]; ok {
		logLine.Level = LogLevel(strings.ToUpper(level))
	} else {
		logLine.Level = DetectLevel(line)
	}

	return logLine
}

// CanParse проверяет, является ли строка logfmt форматом.
// Требуется минимум 2 известных logfmt ключа.
func (p *LogfmtParser) CanParse(line string) bool {
	matches := logfmtPattern.FindAllStringSubmatch(line, -1)
	if len(matches) < 2 {
		return false
	}
	validKeys := 0
	logKeys := map[string]bool{
		"level": true, "msg": true, "message": true, "timestamp": true,
		"time": true, "logger": true, "host": true, "service": true,
		"error": true, "err": true, "status": true, "method": true,
		"severity": true, "@timestamp": true,
	}
	for _, m := range matches {
		if len(m) == 3 {
			key := m[1]
			if logKeys[key] {
				validKeys++
			}
		}
	}
	return validKeys >= 2
}

// PlainParser парсит обычные текстовые логи без определённого формата.
type PlainParser struct{}

// Parse парсит plain text строку в LogLine.
func (p PlainParser) Parse(line string, source Source) *LogLine {
	return &LogLine{
		Source:    source,
		Raw:       line,
		Content:   line,
		IsJSON:    false,
		Parsed:    nil,
		Timestamp: ParseTimestamp(line),
		Level:     DetectLevel(line),
	}
}

// CanParse всегда возвращает true, PlainParser - fallback парсер.
func (p *PlainParser) CanParse(line string) bool {
	return true
}

// RingBuffer представляет потокобезопасный кольцевой буфер для хранения логов.
// При заполнении старые записи вытесняются новыми.
type RingBuffer struct {
	lines []LogLine // Массив хранимых строк
	size  int       // Максимальный размер буфера
	head  int       // Индекс головы (куда будет записана следующая строка)
	count int       // Текущее количество строк в буфере
	mu    sync.RWMutex
}

// NewRingBuffer создаёт новый RingBuffer указанного размера.
// Размер по умолчанию - 5000 строк.
func NewRingBuffer(size int) *RingBuffer {
	if size <= 0 {
		size = 5000
	}
	return &RingBuffer{
		lines: make([]LogLine, size),
		size:  size,
	}
}

// Add добавляет новую строку в буфер.
// Потокобезопасно.
func (rb *RingBuffer) Add(line LogLine) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.lines[rb.head] = line
	rb.head = (rb.head + 1) % rb.size
	if rb.count < rb.size {
		rb.count++
	}
}

// GetAll возвращает все строки из буфера в порядке от старых к новым.
// Потокобезопасно.
func (rb *RingBuffer) GetAll() []LogLine {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.count == 0 {
		return nil
	}
	result := make([]LogLine, rb.count)
	if rb.count < rb.size {
		copy(result, rb.lines[:rb.count])
	} else {
		copy(result, rb.lines[rb.head:])
		copy(result[rb.size-rb.head:], rb.lines[:rb.head])
	}
	return result
}

// GetFiltered возвращает отфильтрованные строки.
// Фильтрация по тексту (без учёта регистра) и/или по источникам.
func (rb *RingBuffer) GetFiltered(filter string, includeSources map[string]bool) []LogLine {
	lines := rb.GetAll()
	if filter == "" && len(includeSources) == 0 {
		return lines
	}
	filtered := make([]LogLine, 0, len(lines))
	for _, line := range lines {
		if len(includeSources) > 0 {
			if !includeSources[line.Source.Path] {
				continue
			}
		}
		if filter != "" {
			lowerContent := strings.ToLower(line.Content)
			lowerFilter := strings.ToLower(filter)
			if !strings.Contains(lowerContent, lowerFilter) {
				continue
			}
		}
		filtered = append(filtered, line)
	}
	return filtered
}

// Len возвращает текущее количество строк в буфере.
func (rb *RingBuffer) Len() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}

// Clear очищает буфер.
func (rb *RingBuffer) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.head = 0
	rb.count = 0
}

// GetLastN возвращает последние N строк из буфера.
func (rb *RingBuffer) GetLastN(n int) []LogLine {
	all := rb.GetAll()
	if len(all) <= n {
		return all
	}
	return all[len(all)-n:]
}

// FuzzyMatch проверяет, содержит ли текст паттерн (без учёта регистра).
func FuzzyMatch(text, pattern string) bool {
	text = strings.ToLower(text)
	pattern = strings.ToLower(pattern)

	if strings.Contains(text, pattern) {
		return true
	}

	return fuzzyMatchRecursive(text, pattern)
}

// fuzzyMatchRecursive рекурсивная реализация fuzzy matching.
func fuzzyMatchRecursive(text, pattern string) bool {
	if pattern == "" {
		return true
	}
	if text == "" {
		return false
	}

	if text[0] == pattern[0] {
		return fuzzyMatchRecursive(text[1:], pattern[1:])
	}
	return fuzzyMatchRecursive(text[1:], pattern)
}

// HighlightMatches возвращает текст с подсветкой совпадений.
// Совпадения обрамляются маркерами \x02 и \x03.
func HighlightMatches(text, pattern string) string {
	if pattern == "" {
		return text
	}

	lowerText := strings.ToLower(text)
	lowerPattern := strings.ToLower(pattern)

	var result strings.Builder
	lastEnd := 0

	for {
		idx := strings.Index(lowerText[lastEnd:], lowerPattern)
		if idx == -1 {
			result.WriteString(text[lastEnd:])
			break
		}

		actualIdx := lastEnd + idx
		result.WriteString(text[lastEnd:actualIdx])
		result.WriteString("\x02")
		result.WriteString(text[actualIdx : actualIdx+len(pattern)])
		result.WriteString("\x03")
		lastEnd = actualIdx + len(pattern)
	}

	return result.String()
}

// ReadExistingContent читает содержимое файла и отправляет строки в канал.
// Используется провайдерами для начального чтения файлов.
func ReadExistingContent(file *os.File, source Source, parser *MultiParser, logChan chan<- LogLine) {
	reader := bufio.NewReader(file)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		line = strings.TrimSuffix(line, "\n")
		line = strings.TrimSuffix(line, "\r")

		if line != "" {
			logLine := parser.Parse(line, source)
			if logLine != nil {
				select {
				case logChan <- *logLine:
				case <-time.After(10 * time.Millisecond):
				}
			}
		}
	}
}
