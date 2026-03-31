package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/spf13/pflag"
	"github.com/turkprogrammer/logt/internal/config"
	"github.com/turkprogrammer/logt/internal/domain"
	"github.com/turkprogrammer/logt/internal/domain/jsonpath"
	"github.com/turkprogrammer/logt/internal/provider"
	"github.com/turkprogrammer/logt/internal/ui"
)

var version = "0.5.0"

func main() {
	// Обработка subcommands (должно быть до парсинга flags)
	if len(os.Args) > 1 && os.Args[1] == "completion" {
		completionCmd(os.Args)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		log.Printf("Warning: failed to load config: %v", err)
		cfg = config.DefaultConfig()
	}

	// Показ версии
	showVersion(cfg)

	if cfg.Path != "" {
		paths := strings.Split(cfg.Path, ",")
		runWithPaths(paths, cfg)
	} else if len(pflag.Args()) > 0 {
		paths := pflag.Args()
		runWithPaths(paths, cfg)
	} else if provider.IsStdinPiped() {
		runStdin(cfg)
	} else {
		showHelp()
	}
}

// showVersion показывает версию и выходит.
func showVersion(cfg *config.Config) {
	if cfg.Headless && cfg.Stats {
		fmt.Printf("LogT v%s\n", version)
	}
}

func runWithPaths(paths []string, cfg *config.Config) {
	var fileProvider provider.Provider
	mp := provider.NewMultiProvider()

	// Используем watcher для Linux/macOS, polling для Windows
	if provider.IsWatcherSupported() && provider.IsWatcherPreferred() {
		fileProvider = provider.NewWatcherProvider()
	} else {
		fileProvider = provider.NewFileProvider()
	}

	mp.AddProvider(fileProvider)

	expandedPaths := provider.ExpandPaths(paths)
	if len(expandedPaths) == 0 {
		log.Fatalf("No files found matching: %v", paths)
	}

	if err := fileProvider.Watch(expandedPaths); err != nil {
		log.Fatalf("Failed to watch files: %v", err)
	}

	run(mp, cfg)
}

func runStdin(cfg *config.Config) {
	stdinProvider := provider.NewStdinProvider()
	mp := provider.NewMultiProvider()
	mp.AddProvider(stdinProvider)

	if err := stdinProvider.Start(); err != nil {
		log.Fatalf("Failed to start stdin provider: %v", err)
	}

	run(mp, cfg)
}

func run(mp *provider.MultiProvider, cfg *config.Config) {
	// Headless режим
	if cfg.Headless {
		runHeadless(mp, cfg)
		return
	}

	if cfg.Forward != "" {
		go startForwarding(mp, cfg.Forward)
	}

	// Парсинг временных фильтров
	var since, until *time.Time
	if cfg.Since != "" {
		t, err := domain.ParseSince(cfg.Since)
		if err != nil {
			log.Printf("Warning: invalid --since value %q: %v", cfg.Since, err)
		} else {
			since = &t
		}
	}
	if cfg.Until != "" {
		t, err := domain.ParseSince(cfg.Until)
		if err != nil {
			log.Printf("Warning: invalid --until value %q: %v", cfg.Until, err)
		} else {
			until = &t
		}
	}

	// Парсинг JSON Path фильтра
	var jsonFilter *jsonpath.Filter
	if cfg.JsonFilter != "" {
		var err error
		jsonFilter, err = jsonpath.Parse(cfg.JsonFilter)
		if err != nil {
			log.Printf("Warning: invalid --json value %q: %v", cfg.JsonFilter, err)
			jsonFilter = nil
		}
	}

	model := ui.NewModel(mp, since, until, jsonFilter)

	p := tea.NewProgram(model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		log.Fatalf("Failed to run UI: %v", err)
	}
}

func showHelp() {
	fmt.Println("LogT — Modern Log Explorer (TUI)")
	fmt.Println("\nUsage: logt [path ...] [flags]")
	fmt.Println("\nFlags:")
	pflag.PrintDefaults()
	fmt.Println("\nExamples:")
	fmt.Println("  logt /var/log/*.log")
	fmt.Println("  logt --path ./logs/*.log --level error")
	fmt.Println("  logt --forward filtered.log ./app.log")
	fmt.Println("  logt ./api/*.log ./db/*.log")
	fmt.Println("  logt --since 1h ./app.log")
	fmt.Println("  logt --since 30m --until 10m ./app.log")
	fmt.Println("  logt --json '.level == \"error\"' ./app.log")
	fmt.Println("  logt --json '.message | startswith(\"Error\")' ./app.log")
	fmt.Println("  cat app.log | logt")
	fmt.Println("\nHeadless mode:")
	fmt.Println("  logt --headless --stats ./app.log")
	fmt.Println("  logt --headless --tail 100 ./app.log")
	fmt.Println("  logt --headless --stats --tail 50 --since 1h ./app.log")
	fmt.Println("\nConfig: ~/.config/logt/config.yaml or ./logt.yaml")
}

func startForwarding(mp *provider.MultiProvider, forwardPath string) {
	var writer io.WriteCloser
	var err error

	if forwardPath == "stdout" || forwardPath == "-" {
		writer = os.Stdout
	} else {
		writer, err = os.OpenFile(forwardPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("Failed to open forward file: %v", err)
			return
		}
		defer writer.Close()
	}

	for logLine := range mp.LogChan() {
		fmt.Fprintln(writer, logLine.Content)
	}
}
