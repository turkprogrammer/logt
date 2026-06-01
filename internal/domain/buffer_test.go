package domain

import (
	"sync"
	"testing"
	"time"
)

func TestRingBuffer_BasicOperations(t *testing.T) {
	rb := NewRingBuffer(10)

	if rb.Len() != 0 {
		t.Errorf("Expected empty buffer, got length %d", rb.Len())
	}

	rb.Add(LogLine{Content: "line1"})
	rb.Add(LogLine{Content: "line2"})

	if rb.Len() != 2 {
		t.Errorf("Expected length 2, got %d", rb.Len())
	}

	lines := rb.GetAll()
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}
}

func TestRingBuffer_Overflow(t *testing.T) {
	size := 100
	rb := NewRingBuffer(size)

	for i := 0; i < 50; i++ {
		rb.Add(LogLine{Content: "early_line"})
	}

	if rb.Len() != 50 {
		t.Errorf("Expected length 50, got %d", rb.Len())
	}

	for i := 0; i < size+50; i++ {
		rb.Add(LogLine{Content: "overflow_line"})
	}

	if rb.Len() != size {
		t.Errorf("Expected length %d after overflow, got %d", size, rb.Len())
	}

	lines := rb.GetAll()
	if len(lines) != size {
		t.Errorf("Expected %d lines from GetAll, got %d", size, len(lines))
	}

	hasOverflowLines := false
	for _, line := range lines {
		if line.Content == "overflow_line" {
			hasOverflowLines = true
			break
		}
	}
	if !hasOverflowLines {
		t.Error("Expected overflow lines to be present (newest data)")
	}

	hasEarlyLines := false
	for _, line := range lines {
		if line.Content == "early_line" {
			hasEarlyLines = true
			break
		}
	}
	if hasEarlyLines {
		t.Error("Early lines should have been evicted after overflow")
	}
}

func TestRingBuffer_ExactSize(t *testing.T) {
	size := 5
	rb := NewRingBuffer(size)

	for i := 0; i < size; i++ {
		rb.Add(LogLine{Content: "line"})
	}

	lines := rb.GetAll()
	if len(lines) != size {
		t.Errorf("Expected exactly %d lines, got %d", size, len(lines))
	}

	for i := 0; i < 10; i++ {
		rb.Add(LogLine{Content: "extra"})
	}

	lines = rb.GetAll()
	if len(lines) != size {
		t.Errorf("Expected exactly %d lines after overflow, got %d", size, len(lines))
	}
}

func TestRingBuffer_EmptyBuffer(t *testing.T) {
	rb := NewRingBuffer(10)

	lines := rb.GetAll()
	if len(lines) != 0 {
		t.Errorf("Expected empty slice for empty buffer, got %v", lines)
	}
}

func TestRingBuffer_GetLastN(t *testing.T) {
	rb := NewRingBuffer(100)

	for i := 0; i < 50; i++ {
		rb.Add(LogLine{Content: "line"})
	}

	last10 := rb.GetLastN(10)
	if len(last10) != 10 {
		t.Errorf("Expected 10 lines, got %d", len(last10))
	}

	last100 := rb.GetLastN(100)
	if len(last100) != 50 {
		t.Errorf("Expected 50 lines (total), got %d", len(last100))
	}
}

func TestRingBuffer_ConcurrentAccess(t *testing.T) {
	rb := NewRingBuffer(1000)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				rb.Add(LogLine{Content: "concurrent_line"})
				time.Sleep(time.Microsecond)
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 100; j++ {
			rb.GetAll()
			rb.Len()
			time.Sleep(time.Microsecond)
		}
	}()

	wg.Wait()

	if rb.Len() > 1000 {
		t.Errorf("Buffer exceeded capacity: got %d, max %d", rb.Len(), 1000)
	}
}

func TestRingBuffer_MemoryBound(t *testing.T) {
	rb := NewRingBuffer(5000)

	for i := 0; i < 100000; i++ {
		rb.Add(LogLine{Content: "x"})
	}

	if rb.Len() != 5000 {
		t.Errorf("Memory bound violated: expected 5000, got %d", rb.Len())
	}

	memEstimate := rb.Len() * 200
	if memEstimate > 2*1024*1024 {
		t.Logf("Memory usage estimate: ~%d bytes for 5000 lines", memEstimate)
	}
}
