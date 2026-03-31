// Package domain тестирует калькулятор скорости (rate).
package domain

import (
	"testing"
	"time"
)

func TestRateCalculator_Basic(t *testing.T) {
	rc := NewRateCalculator()

	// Добавляем 10000 строк для надёжности
	for i := 0; i < 10000; i++ {
		rc.Update()
	}

	// Ждём 500ms для расчёта elapsed time
	time.Sleep(500 * time.Millisecond)

	rate := rc.Rate()
	t.Logf("rate=%f", rate)

	if rate <= 0 {
		t.Errorf("Expected positive rate, got %f", rate)
	}
}

func TestRateCalculator_Empty(t *testing.T) {
	rc := NewRateCalculator()

	rate := rc.Rate()
	if rate != 0 {
		t.Errorf("Expected rate=0 for empty calculator, got %f", rate)
	}
}

func TestRateCalculator_Reset(t *testing.T) {
	rc := NewRateCalculator()

	// Добавляем 50 строк
	for i := 0; i < 50; i++ {
		rc.Update()
	}

	rc.Reset()

	rate := rc.Rate()
	if rate != 0 {
		t.Errorf("Expected rate=0 after reset, got %f", rate)
	}

	count := rc.Count()
	if count != 0 {
		t.Errorf("Expected count=0 after reset, got %d", count)
	}
}

func TestRateCalculator_Count(t *testing.T) {
	rc := NewRateCalculator()

	if rc.Count() != 0 {
		t.Errorf("Expected count=0 initially, got %d", rc.Count())
	}

	for i := 0; i < 25; i++ {
		rc.Update()
	}

	if rc.Count() != 25 {
		t.Errorf("Expected count=25, got %d", rc.Count())
	}
}

func TestRateCalculator_RateStability(t *testing.T) {
	rc := NewRateCalculator()

	// Добавляем 10000 строк
	for i := 0; i < 10000; i++ {
		rc.Update()
	}

	// Ждём 300ms для расчёта elapsed time
	time.Sleep(300 * time.Millisecond)

	// Получаем rate несколько раз
	rate1 := rc.Rate()
	time.Sleep(100 * time.Millisecond)
	rate2 := rc.Rate()

	// Rate должен быть положительным и стабильным
	if rate1 <= 0 {
		t.Errorf("Expected positive rate1, got %f", rate1)
	}
	if rate2 <= 0 {
		t.Errorf("Expected positive rate2, got %f", rate2)
	}
}
