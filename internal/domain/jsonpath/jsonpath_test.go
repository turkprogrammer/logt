// Package jsonpath предоставляет парсер и исполнитель JSON Path фильтров.
// Поддерживаемые операторы: ==, !=, startswith, contains
package jsonpath

import (
	"testing"
)

func TestParseJsonPath_Equals(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		wantPath string
		wantOp   Operator
		wantVal  string
		wantErr  bool
	}{
		{
			name:     "simple equals string",
			expr:     `.level == "error"`,
			wantPath: "level",
			wantOp:   OpEquals,
			wantVal:  "error",
			wantErr:  false,
		},
		{
			name:     "simple equals number",
			expr:     `.status == 500`,
			wantPath: "status",
			wantOp:   OpEquals,
			wantVal:  "500",
			wantErr:  false,
		},
		{
			name:     "simple equals boolean",
			expr:     `.success == true`,
			wantPath: "success",
			wantOp:   OpEquals,
			wantVal:  "true",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Parse(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if filter.Path != tt.wantPath {
					t.Errorf("Path = %v, want %v", filter.Path, tt.wantPath)
				}
				if filter.Operator != tt.wantOp {
					t.Errorf("Operator = %v, want %v", filter.Operator, tt.wantOp)
				}
				if filter.Value != tt.wantVal {
					t.Errorf("Value = %v, want %v", filter.Value, tt.wantVal)
				}
			}
		})
	}
}

func TestParseJsonPath_NotEquals(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		wantPath string
		wantOp   Operator
		wantVal  string
		wantErr  bool
	}{
		{
			name:     "not equals string",
			expr:     `.level != "debug"`,
			wantPath: "level",
			wantOp:   OpNotEquals,
			wantVal:  "debug",
			wantErr:  false,
		},
		{
			name:     "not equals number",
			expr:     `.code != 404`,
			wantPath: "code",
			wantOp:   OpNotEquals,
			wantVal:  "404",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Parse(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if filter.Path != tt.wantPath {
					t.Errorf("Path = %v, want %v", filter.Path, tt.wantPath)
				}
				if filter.Operator != tt.wantOp {
					t.Errorf("Operator = %v, want %v", filter.Operator, tt.wantOp)
				}
				if filter.Value != tt.wantVal {
					t.Errorf("Value = %v, want %v", filter.Value, tt.wantVal)
				}
			}
		})
	}
}

func TestParseJsonPath_StartsWith(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		wantPath string
		wantOp   Operator
		wantVal  string
		wantErr  bool
	}{
		{
			name:     "startswith string",
			expr:     `.message | startswith("Error")`,
			wantPath: "message",
			wantOp:   OpStartsWith,
			wantVal:  "Error",
			wantErr:  false,
		},
		{
			name:     "startswith path",
			expr:     `.url | startswith("/api")`,
			wantPath: "url",
			wantOp:   OpStartsWith,
			wantVal:  "/api",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Parse(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if filter.Path != tt.wantPath {
					t.Errorf("Path = %v, want %v", filter.Path, tt.wantPath)
				}
				if filter.Operator != tt.wantOp {
					t.Errorf("Operator = %v, want %v", filter.Operator, tt.wantOp)
				}
				if filter.Value != tt.wantVal {
					t.Errorf("Value = %v, want %v", filter.Value, tt.wantVal)
				}
			}
		})
	}
}

func TestParseJsonPath_Contains(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		wantPath string
		wantOp   Operator
		wantVal  string
		wantErr  bool
	}{
		{
			name:     "contains string",
			expr:     `.message | contains("timeout")`,
			wantPath: "message",
			wantOp:   OpContains,
			wantVal:  "timeout",
			wantErr:  false,
		},
		{
			name:     "contains error",
			expr:     `.error | contains("connection")`,
			wantPath: "error",
			wantOp:   OpContains,
			wantVal:  "connection",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := Parse(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if filter.Path != tt.wantPath {
					t.Errorf("Path = %v, want %v", filter.Path, tt.wantPath)
				}
				if filter.Operator != tt.wantOp {
					t.Errorf("Operator = %v, want %v", filter.Operator, tt.wantOp)
				}
				if filter.Value != tt.wantVal {
					t.Errorf("Value = %v, want %v", filter.Value, tt.wantVal)
				}
			}
		})
	}
}

func TestParseJsonPath_Invalid(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"empty expression", ""},
		{"no operator", ".level"},
		{"invalid operator", `.level === "error"`},
		{"missing value", `.level ==`},
		{"missing path", `== "error"`},
		{"invalid syntax", `level error`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.expr)
			if err == nil {
				t.Errorf("Parse() expected error for %q, got nil", tt.expr)
			}
		})
	}
}

func TestExecuteFilter_Equals(t *testing.T) {
	tests := []struct {
		name   string
		filter *Filter
		data   map[string]any
		want   bool
	}{
		{
			name: "match string",
			filter: &Filter{
				Path:     "level",
				Operator: OpEquals,
				Value:    "error",
			},
			data: map[string]any{
				"level": "error",
			},
			want: true,
		},
		{
			name: "match number",
			filter: &Filter{
				Path:     "status",
				Operator: OpEquals,
				Value:    "500",
			},
			data: map[string]any{
				"status": 500,
			},
			want: true,
		},
		{
			name: "no match string",
			filter: &Filter{
				Path:     "level",
				Operator: OpEquals,
				Value:    "error",
			},
			data: map[string]any{
				"level": "info",
			},
			want: false,
		},
		{
			name: "match boolean",
			filter: &Filter{
				Path:     "success",
				Operator: OpEquals,
				Value:    "true",
			},
			data: map[string]any{
				"success": true,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Execute(tt.filter, tt.data)
			if got != tt.want {
				t.Errorf("Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecuteFilter_NestedPath(t *testing.T) {
	tests := []struct {
		name   string
		filter *Filter
		data   map[string]any
		want   bool
	}{
		{
			name: "nested path match",
			filter: &Filter{
				Path:     "user.name",
				Operator: OpEquals,
				Value:    "admin",
			},
			data: map[string]any{
				"user": map[string]any{
					"name": "admin",
				},
			},
			want: true,
		},
		{
			name: "nested path no match",
			filter: &Filter{
				Path:     "user.name",
				Operator: OpEquals,
				Value:    "guest",
			},
			data: map[string]any{
				"user": map[string]any{
					"name": "admin",
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Execute(tt.filter, tt.data)
			if got != tt.want {
				t.Errorf("Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecuteFilter_MissingField(t *testing.T) {
	filter := &Filter{
		Path:     "level",
		Operator: OpEquals,
		Value:    "error",
	}

	data := map[string]any{
		"message": "test",
	}

	got := Execute(filter, data)
	if got != false {
		t.Errorf("Execute() = %v, want false (missing field)", got)
	}
}

func TestExecuteFilter_NotEquals(t *testing.T) {
	tests := []struct {
		name   string
		filter *Filter
		data   map[string]any
		want   bool
	}{
		{
			name: "not equals match",
			filter: &Filter{
				Path:     "level",
				Operator: OpNotEquals,
				Value:    "debug",
			},
			data: map[string]any{
				"level": "error",
			},
			want: true,
		},
		{
			name: "not equals no match",
			filter: &Filter{
				Path:     "level",
				Operator: OpNotEquals,
				Value:    "error",
			},
			data: map[string]any{
				"level": "error",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Execute(tt.filter, tt.data)
			if got != tt.want {
				t.Errorf("Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecuteFilter_StartsWith(t *testing.T) {
	tests := []struct {
		name   string
		filter *Filter
		data   map[string]any
		want   bool
	}{
		{
			name: "startswith match",
			filter: &Filter{
				Path:     "message",
				Operator: OpStartsWith,
				Value:    "Error",
			},
			data: map[string]any{
				"message": "Error: connection failed",
			},
			want: true,
		},
		{
			name: "startswith no match",
			filter: &Filter{
				Path:     "message",
				Operator: OpStartsWith,
				Value:    "Warning",
			},
			data: map[string]any{
				"message": "Error: connection failed",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Execute(tt.filter, tt.data)
			if got != tt.want {
				t.Errorf("Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecuteFilter_Contains(t *testing.T) {
	tests := []struct {
		name   string
		filter *Filter
		data   map[string]any
		want   bool
	}{
		{
			name: "contains match",
			filter: &Filter{
				Path:     "message",
				Operator: OpContains,
				Value:    "timeout",
			},
			data: map[string]any{
				"message": "connection timeout occurred",
			},
			want: true,
		},
		{
			name: "contains no match",
			filter: &Filter{
				Path:     "message",
				Operator: OpContains,
				Value:    "success",
			},
			data: map[string]any{
				"message": "connection timeout occurred",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Execute(tt.filter, tt.data)
			if got != tt.want {
				t.Errorf("Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}
