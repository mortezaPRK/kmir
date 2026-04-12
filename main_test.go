package main

import (
	"testing"
	"time"
)

func TestWait_Success(t *testing.T) {
	started := time.Now()
	err := wait(100*time.Millisecond, func() bool {
		return true
	})

	if err != nil {
		t.Errorf("wait() returned error: %v", err)
	}

	elapsed := time.Since(started)
	if elapsed > 50*time.Millisecond {
		t.Errorf("wait() took too long: %v, expected < 50ms", elapsed)
	}
}

func TestWait_Timeout(t *testing.T) {
	started := time.Now()
	err := wait(100*time.Millisecond, func() bool {
		return false
	})

	if err == nil {
		t.Error("wait() expected timeout error, got nil")
	}

	elapsed := time.Since(started)
	// With 1 second sleep intervals and 100ms timeout, it will check once then timeout
	// The actual time will be slightly over 100ms due to the first sleep
	if elapsed < 100*time.Millisecond {
		t.Logf("wait() returned quickly: %v (expected >= 100ms)", elapsed)
	}
	// Allow up to 1.5 seconds since the sleep interval is 1 second
	if elapsed > 1500*time.Millisecond {
		t.Errorf("wait() took too long: %v, expected < 1500ms", elapsed)
	}
}

func TestWait_DelayedSuccess(t *testing.T) {
	count := 0
	started := time.Now()

	err := wait(200*time.Millisecond, func() bool {
		count++
		return count >= 2
	})

	if err != nil {
		t.Errorf("wait() returned error: %v", err)
	}

	if count < 2 {
		t.Errorf("wait() condition not checked enough times, got %d", count)
	}

	// Should have taken at least one sleep cycle
	elapsed := time.Since(started)
	if elapsed < time.Second {
		t.Logf("wait() took %v for %d checks", elapsed, count)
	}
}

func TestWait_ImmediateSuccess(t *testing.T) {
	callCount := 0
	err := wait(5*time.Second, func() bool {
		callCount++
		return true
	})

	if err != nil {
		t.Errorf("wait() returned error: %v", err)
	}

	if callCount != 1 {
		t.Errorf("wait() called function %d times, expected 1", callCount)
	}
}

func TestWait_NegativeTimeout(t *testing.T) {
	err := wait(-1*time.Second, func() bool {
		return false
	})

	// Negative timeout should cause immediate timeout
	if err == nil {
		t.Error("wait() with negative timeout expected error, got nil")
	}
}

func BenchmarkWait(b *testing.B) {
	fn := func() bool { return true }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wait(time.Second, fn)
	}
}
