// Package domain предоставляет калькулятор скорости (rate) для логов.
package domain

import (
	"sync"
	"time"
)

// RateCalculator рассчитывает скорость поступления логов (lines/sec).
type RateCalculator struct {
	mu        sync.RWMutex
	count     int
	startTime time.Time
	lastRate  float64
	lastCalc  time.Time
}

// NewRateCalculator создаёт новый RateCalculator.
func NewRateCalculator() *RateCalculator {
	return &RateCalculator{
		startTime: time.Now(),
		lastCalc:  time.Now(),
	}
}

// Update инкрементирует счётчик строк.
func (rc *RateCalculator) Update() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.count++
	rc.calculateRateLocked()
}

// Rate возвращает текущую скорость (lines/sec).
func (rc *RateCalculator) Rate() float64 {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Пересчитываем rate каждый раз
	elapsed := time.Since(rc.startTime).Seconds()
	if elapsed > 0 {
		rc.lastRate = float64(rc.count) / elapsed
	} else {
		rc.lastRate = 0
	}

	return rc.lastRate
}

// Count возвращает общее количество строк.
func (rc *RateCalculator) Count() int {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.count
}

// Reset сбрасывает счётчики.
func (rc *RateCalculator) Reset() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.count = 0
	rc.startTime = time.Now()
	rc.lastRate = 0
	rc.lastCalc = time.Now()
}

// calculateRateLocked рассчитывает rate (должна вызываться с захваченным lock).
func (rc *RateCalculator) calculateRateLocked() float64 {
	elapsed := time.Since(rc.startTime).Seconds()
	if elapsed > 0 {
		rc.lastRate = float64(rc.count) / elapsed
	} else {
		rc.lastRate = 0
	}
	rc.lastCalc = time.Now()
	return rc.lastRate
}
