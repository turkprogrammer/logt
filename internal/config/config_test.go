package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.BufferSize != 5000 {
		t.Errorf("Expected BufferSize 5000, got %d", cfg.BufferSize)
	}
	if cfg.BufferMax != 10000 {
		t.Errorf("Expected BufferMax 10000, got %d", cfg.BufferMax)
	}
	if cfg.Theme != "dark" {
		t.Errorf("Expected Theme 'dark', got %s", cfg.Theme)
	}
}

func TestSourcesFromConfig_Path(t *testing.T) {
	cfg := &Config{
		Path: "app.log,debug.log",
	}

	sources := cfg.SourcesFromConfig()
	if len(sources) != 2 {
		t.Errorf("Expected 2 sources, got %d", len(sources))
	}
}

func TestSourcesFromConfig_Sources(t *testing.T) {
	cfg := &Config{
		Sources: []string{"/var/log/*.log"},
	}

	sources := cfg.SourcesFromConfig()
	if len(sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(sources))
	}
}

func TestSourcesFromConfig_Combined(t *testing.T) {
	cfg := &Config{
		Path:    "app.log",
		Sources: []string{"/var/log/*.log"},
	}

	sources := cfg.SourcesFromConfig()
	if len(sources) != 2 {
		t.Errorf("Expected 2 sources, got %d", len(sources))
	}
}

func TestLoad_NoConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.BufferSize != 5000 {
		t.Errorf("Expected default BufferSize 5000, got %d", cfg.BufferSize)
	}
}

func TestLoad_BufferSizeValidation(t *testing.T) {
	cfg := &Config{
		BufferSize: 0,
		BufferMax:  0,
	}

	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 5000
	}
	if cfg.BufferMax <= 0 {
		cfg.BufferMax = 10000
	}

	if cfg.BufferSize != 5000 {
		t.Errorf("Expected BufferSize 5000 after validation, got %d", cfg.BufferSize)
	}
	if cfg.BufferMax != 10000 {
		t.Errorf("Expected BufferMax 10000 after validation, got %d", cfg.BufferMax)
	}
}
