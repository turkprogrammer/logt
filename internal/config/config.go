// Package config реализует конфигурацию приложения LogT.
//
// Поддерживает:
//   - YAML файлы конфигурации (~/.config/logt/config.yaml)
//   - Переменные окружения (LOGT_*)
//   - Флаги командной строки
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config представляет конфигурацию приложения.
type Config struct {
	Path       string   `mapstructure:"path"`
	Level      string   `mapstructure:"level"`
	BufferSize int      `mapstructure:"buffer-size"`
	BufferMax  int      `mapstructure:"buffer-max"`
	Theme      string   `mapstructure:"theme"`
	Forward    string   `mapstructure:"forward"`
	Sources    []string `mapstructure:"sources"`
	Since      string   `mapstructure:"since"`
	Until      string   `mapstructure:"until"`
	JsonFilter string   `mapstructure:"json-filter"`
	Headless   bool     `mapstructure:"headless"`
	Tail       int      `mapstructure:"tail"`
	Stats      bool     `mapstructure:"stats"`
	Export     string   `mapstructure:"export"`
}

// DefaultConfig возвращает конфигурацию по умолчанию.
func DefaultConfig() *Config {
	return &Config{
		BufferSize: 5000,
		BufferMax:  10000,
		Theme:      "dark",
		Level:      "",
		Path:       "",
		Forward:    "",
		Sources:    []string{},
		Headless:   false,
		Tail:       0, // 0 = все строки
		Stats:      false,
		Export:     "",
	}
}

// Load загружает конфигурацию из файлов, переменных окружения и флагов.
func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err == nil {
		configDir := filepath.Join(home, ".config", "logt")
		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.SetConfigName("logt")
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")

	viper.SetEnvPrefix("LOGT")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	pflag.CommandLine.StringP("path", "p", "", "Пути к файлам или glob паттерны")
	pflag.CommandLine.StringP("level", "l", "", "Фильтр по уровню (debug,info,warn,error)")
	pflag.CommandLine.IntP("buffer", "b", 5000, "Размер буфера")
	pflag.CommandLine.IntP("max-buffer", "m", 10000, "Максимальный размер буфера")
	pflag.CommandLine.StringP("theme", "t", "dark", "Тема (dark, light)")
	pflag.CommandLine.StringP("forward", "f", "", "Экспорт логов (файл или stdout)")
	pflag.CommandLine.StringP("since", "S", "", "Фильтр с времени (1h, 30m, 2024-01-15)")
	pflag.CommandLine.StringP("until", "U", "", "Фильтр по время (1h, 30m, 2024-01-15)")
	pflag.CommandLine.StringP("json", "j", "", "JSON Path фильтр (например: '.level == \"error\"')")
	pflag.CommandLine.BoolP("headless", "H", false, "Режим без TUI (CLI)")
	pflag.CommandLine.IntP("tail", "n", 0, "Последние N строк (0 = все)")
	pflag.CommandLine.BoolP("stats", "s", false, "Вывод статистики")
	pflag.CommandLine.StringP("export", "e", "", "Экспорт bookmarks в файл")
	pflag.CommandLine.BoolP("version", "v", false, "Версия")
	pflag.CommandLine.BoolP("help", "h", false, "Помощь")

	viper.BindPFlag("path", pflag.CommandLine.Lookup("path"))
	viper.BindPFlag("level", pflag.CommandLine.Lookup("level"))
	viper.BindPFlag("buffer-size", pflag.CommandLine.Lookup("buffer"))
	viper.BindPFlag("buffer-max", pflag.CommandLine.Lookup("max-buffer"))
	viper.BindPFlag("theme", pflag.CommandLine.Lookup("theme"))
	viper.BindPFlag("forward", pflag.CommandLine.Lookup("forward"))
	viper.BindPFlag("since", pflag.CommandLine.Lookup("since"))
	viper.BindPFlag("until", pflag.CommandLine.Lookup("until"))
	viper.BindPFlag("json-filter", pflag.CommandLine.Lookup("json"))
	viper.BindPFlag("headless", pflag.CommandLine.Lookup("headless"))
	viper.BindPFlag("tail", pflag.CommandLine.Lookup("tail"))
	viper.BindPFlag("stats", pflag.CommandLine.Lookup("stats"))
	viper.BindPFlag("export", pflag.CommandLine.Lookup("export"))

	pflag.Parse()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Printf("Используется конфигурация: %s\n", viper.ConfigFileUsed())
	}

	cfg := &Config{
		Path:       viper.GetString("path"),
		Level:      viper.GetString("level"),
		BufferSize: viper.GetInt("buffer-size"),
		BufferMax:  viper.GetInt("buffer-max"),
		Theme:      viper.GetString("theme"),
		Forward:    viper.GetString("forward"),
		Sources:    viper.GetStringSlice("sources"),
		Since:      viper.GetString("since"),
		Until:      viper.GetString("until"),
		JsonFilter: viper.GetString("json-filter"),
		Headless:   viper.GetBool("headless"),
		Tail:       viper.GetInt("tail"),
		Stats:      viper.GetBool("stats"),
		Export:     viper.GetString("export"),
	}

	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 5000
	}
	if cfg.BufferMax <= 0 {
		cfg.BufferMax = 10000
	}

	return cfg, nil
}

// Sources возвращает список источников из конфигурации.
func (c *Config) SourcesFromConfig() []string {
	var sources []string

	if c.Path != "" {
		sources = append(sources, strings.Split(c.Path, ",")...)
	}

	if len(c.Sources) > 0 {
		sources = append(sources, c.Sources...)
	}

	return sources
}
