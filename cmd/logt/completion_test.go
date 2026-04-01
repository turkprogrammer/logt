package main

import (
	"testing"
)

func TestGenBash(t *testing.T) {
	script := genBash()

	if len(script) == 0 {
		t.Error("Expected non-empty bash completion script")
	}

	// Проверяем ключевые элементы
	if !contains(script, "_logt()") {
		t.Error("Expected _logt() function in bash script")
	}

	if !contains(script, "complete -F _logt logt") {
		t.Error("Expected complete -F _logt logt in bash script")
	}
}

func TestGenZsh(t *testing.T) {
	script := genZsh()

	if len(script) == 0 {
		t.Error("Expected non-empty zsh completion script")
	}

	// Проверяем ключевые элементы
	if !contains(script, "#compdef logt") {
		t.Error("Expected #compdef logt in zsh script")
	}

	if !contains(script, "_arguments") {
		t.Error("Expected _arguments in zsh script")
	}
}

func TestGenFish(t *testing.T) {
	script := genFish()

	if len(script) == 0 {
		t.Error("Expected non-empty fish completion script")
	}

	// Проверяем ключевые элементы
	if !contains(script, "complete -c logt") {
		t.Error("Expected complete -c logt in fish script")
	}
}

func TestRunCompletion_Invalid(t *testing.T) {
	err := runCompletion("invalid")
	if err == nil {
		t.Error("Expected error for invalid shell")
	}
}

func TestRunCompletion_Bash(t *testing.T) {
	err := runCompletion("bash")
	if err != nil {
		t.Errorf("Unexpected error for bash: %v", err)
	}
}

func TestRunCompletion_Zsh(t *testing.T) {
	err := runCompletion("zsh")
	if err != nil {
		t.Errorf("Unexpected error for zsh: %v", err)
	}
}

func TestRunCompletion_Fish(t *testing.T) {
	err := runCompletion("fish")
	if err != nil {
		t.Errorf("Unexpected error for fish: %v", err)
	}
}

// contains проверяет, содержит ли строка подстроку.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
