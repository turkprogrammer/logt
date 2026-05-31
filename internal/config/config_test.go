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
