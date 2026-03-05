package vm

import (
	"testing"
	"time"
)

func TestGCRateLimitSkipsRepeatedCalls(t *testing.T) {
	v := New(WithLimits(Limits{MinGCInterval: time.Second}))
	for i := 0; i < 100; i++ {
		v.ProcessGcFinalizers()
	}
	if v.gcCallCount != 1 {
		t.Errorf("expected 1 GC call with rate limiting, got %d", v.gcCallCount)
	}
}

func TestGCRateLimitAllowsAfterInterval(t *testing.T) {
	v := New(WithLimits(Limits{MinGCInterval: time.Millisecond}))
	v.ProcessGcFinalizers()
	if v.gcCallCount != 1 {
		t.Fatalf("expected 1 GC call, got %d", v.gcCallCount)
	}
	time.Sleep(2 * time.Millisecond)
	v.ProcessGcFinalizers()
	if v.gcCallCount != 2 {
		t.Errorf("expected 2 GC calls after interval, got %d", v.gcCallCount)
	}
}

func TestGCDisabled(t *testing.T) {
	v := New(WithLimits(Limits{MinGCInterval: -1}))
	for i := 0; i < 100; i++ {
		v.ProcessGcFinalizers()
	}
	if v.gcCallCount != 0 {
		t.Errorf("expected 0 GC calls when disabled, got %d", v.gcCallCount)
	}
}

func TestGCNoRateLimit(t *testing.T) {
	v := New(WithLimits(Limits{MinGCInterval: 0}))
	for i := 0; i < 5; i++ {
		v.ProcessGcFinalizers()
	}
	if v.gcCallCount != 5 {
		t.Errorf("expected 5 GC calls with no rate limit, got %d", v.gcCallCount)
	}
}
