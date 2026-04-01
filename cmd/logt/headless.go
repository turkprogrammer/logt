package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/turkprogrammer/logt/internal/config"
	"github.com/turkprogrammer/logt/internal/domain"
	"github.com/turkprogrammer/logt/internal/provider"
)

// runHeadless запускает headless режим (без TUI).
func runHeadless(mp *provider.MultiProvider, cfg *config.Config) {
	// Ждём немного для чтения данных
	time.Sleep(100 * time.Millisecond)

	// Закрываем провайдер
	mp.Close()

	// Вывод статистики
	if cfg.Stats {
		printStats(mp.Buffer())
	}

	// Вывод последних N строк
	if cfg.Tail > 0 || !cfg.Stats {
		printTail(mp.Buffer(), cfg.Tail, cfg.Export)
	}
}

// printStats выводит статистику логов.
func printStats(buffer *domain.RingBuffer) {
	stats := buffer.CalculateStats()
	fmt.Print(stats.String())
	fmt.Println()
}

// printTail выводит последние N строк.
func printTail(buffer *domain.RingBuffer, tail int, exportPath string) {
	var lines []domain.LogLine

	if tail > 0 {
		lines = buffer.GetLastN(tail)
	} else {
		lines = buffer.GetAll()
	}

	var writer io.WriteCloser = os.Stdout
	if exportPath != "" {
		var err error
		writer, err = os.OpenFile(exportPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("Failed to open export file: %v", err)
			return
		}
		defer writer.Close()
	}

	for _, line := range lines {
		fmt.Fprintln(writer, line.Content)
	}
}
