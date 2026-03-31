// Package jsonpath предоставляет парсер и исполнитель JSON Path фильтров.
// Поддерживаемые операторы: ==, !=, startswith, contains
package jsonpath

import (
	"fmt"
	"strconv"
	"strings"
)

// Operator представляет тип оператора сравнения.
type Operator int

// Константы операторов.
const (
	OpEquals Operator = iota
	OpNotEquals
	OpStartsWith
	OpContains
)

// String возвращает строковое представление оператора.
func (op Operator) String() string {
	switch op {
	case OpEquals:
		return "=="
	case OpNotEquals:
		return "!="
	case OpStartsWith:
		return "startswith"
	case OpContains:
		return "contains"
	default:
		return "unknown"
	}
}

// Filter представляет распарсенный JSON Path фильтр.
type Filter struct {
	Path     string   // Путь к полю (например, "level" или "user.name")
	Operator Operator // Оператор сравнения
	Value    string   // Значение для сравнения
}

// Parse парсит JSON Path выражение в Filter.
// Поддерживаемые форматы:
//   - .field == "value"
//   - .field != "value"
//   - .field | startswith("value")
//   - .field | contains("value")
//
// Примеры:
//
//	Parse(`.level == "error"`) → &Filter{Path: "level", Operator: OpEquals, Value: "error"}
//	Parse(`.message | startswith("Error")`) → &Filter{Path: "message", Operator: OpStartsWith, Value: "Error"}
func Parse(expr string) (*Filter, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("empty expression")
	}

	// Проверяем операторы в порядке приоритета
	switch {
	case strings.Contains(expr, " | startswith("):
		return parsePipeFunction(expr, "startswith", OpStartsWith)
	case strings.Contains(expr, " | contains("):
		return parsePipeFunction(expr, "contains", OpContains)
	case strings.Contains(expr, " == "):
		return parseComparison(expr, "==", OpEquals)
	case strings.Contains(expr, " != "):
		return parseComparison(expr, "!=", OpNotEquals)
	default:
		return nil, fmt.Errorf("invalid expression: no operator found")
	}
}

// parseComparison парсит выражения вида .field op "value".
func parseComparison(expr, op string, operator Operator) (*Filter, error) {
	parts := strings.Split(expr, op)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid expression: expected format '.field %s value'", op)
	}

	path := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	// Извлекаем путь (убираем ведущую точку)
	if !strings.HasPrefix(path, ".") {
		return nil, fmt.Errorf("invalid path: must start with '.'")
	}
	path = strings.TrimPrefix(path, ".")

	// Парсим значение (убираем кавычки если есть)
	value, err := parseValue(value)
	if err != nil {
		return nil, err
	}

	return &Filter{
		Path:     path,
		Operator: operator,
		Value:    value,
	}, nil
}

// parsePipeFunction парсит выражения вида .field | func("value").
func parsePipeFunction(expr, fnName string, operator Operator) (*Filter, error) {
	// Разделяем по " | "
	parts := strings.SplitN(expr, " | ", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid expression: expected format '.field | %s(\"value\")'", fnName)
	}

	path := strings.TrimSpace(parts[0])
	funcCall := strings.TrimSpace(parts[1])

	// Извлекаем путь (убираем ведущую точку)
	if !strings.HasPrefix(path, ".") {
		return nil, fmt.Errorf("invalid path: must start with '.'")
	}
	path = strings.TrimPrefix(path, ".")

	// Парсим функцию: func("value")
	if !strings.HasPrefix(funcCall, fnName+"(") {
		return nil, fmt.Errorf("invalid function: expected %s(...)", fnName)
	}
	if !strings.HasSuffix(funcCall, ")") {
		return nil, fmt.Errorf("invalid function: missing closing parenthesis")
	}

	// Извлекаем аргумент функции
	arg := funcCall[len(fnName)+1 : len(funcCall)-1]
	value, err := parseValue(arg)
	if err != nil {
		return nil, err
	}

	return &Filter{
		Path:     path,
		Operator: operator,
		Value:    value,
	}, nil
}

// parseValue парсит значение (строка в кавычках, число, boolean).
func parseValue(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("empty value")
	}

	// Строка в кавычках
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		return s[1 : len(s)-1], nil
	}

	// Boolean
	if s == "true" || s == "false" {
		return s, nil
	}

	// Число (конвертируем в строку)
	if _, err := strconv.Atoi(s); err == nil {
		return s, nil
	}
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return s, nil
	}

	// Возвращаем как есть (для совместимости)
	return s, nil
}

// Execute применяет фильтр к данным и возвращает результат.
// Возвращает false если поле не найдено или тип не совпадает.
func Execute(filter *Filter, data map[string]any) bool {
	if filter == nil || data == nil {
		return false
	}

	// Извлекаем значение по пути
	value := getValueByPath(data, filter.Path)
	if value == nil {
		return false
	}

	// Применяем оператор
	switch filter.Operator {
	case OpEquals:
		return equals(value, filter.Value)
	case OpNotEquals:
		return !equals(value, filter.Value)
	case OpStartsWith:
		return startsWith(value, filter.Value)
	case OpContains:
		return contains(value, filter.Value)
	default:
		return false
	}
}

// getValueByPath извлекает значение из map по пути (например, "user.name").
func getValueByPath(data map[string]any, path string) any {
	parts := strings.Split(path, ".")
	current := data

	for i, part := range parts {
		val, ok := current[part]
		if !ok {
			return nil
		}

		// Если это последний элемент пути — возвращаем значение
		if i == len(parts)-1 {
			return val
		}

		// Иначе спускаемся глубже (ожидаем map)
		nested, ok := val.(map[string]any)
		if !ok {
			return nil
		}
		current = nested
	}

	return nil
}

// equals сравнивает значение с фильтром.
func equals(value any, filterValue string) bool {
	switch v := value.(type) {
	case string:
		return v == filterValue
	case int:
		return strconv.Itoa(v) == filterValue
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64) == filterValue
	case bool:
		return strconv.FormatBool(v) == filterValue
	default:
		return fmt.Sprintf("%v", v) == filterValue
	}
}

// startsWith проверяет, начинается ли строка с указанного префикса.
func startsWith(value any, prefix string) bool {
	s, ok := value.(string)
	if !ok {
		return false
	}
	return strings.HasPrefix(s, prefix)
}

// contains проверяет, содержит ли строка подстроку.
func contains(value any, substr string) bool {
	s, ok := value.(string)
	if !ok {
		return false
	}
	return strings.Contains(s, substr)
}
