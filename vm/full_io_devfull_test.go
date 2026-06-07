package vm

import (
	"context"
	"os"
	"testing"
)

// TestFullFileFlushResetOnDevFull verifies the /dev/full flush-parity fix:
// after a flush failure (ENOSPC from /dev/full) the underlying bufio.Writer is
// reset so a subsequent small write into freed buffer space does NOT re-return
// Go's sticky flush error — matching C stdio's fflush-then-reuse behavior.
func TestFullFileFlushResetOnDevFull(t *testing.T) {
	if _, err := os.Stat("/dev/full"); err != nil {
		t.Skip("/dev/full not available on this host")
	}

	p := NewTestIoProvider()
	ctx := context.Background()

	f, err := p.Open(ctx, "/dev/full", "w")
	if err != nil {
		t.Fatalf("open /dev/full: %v", err)
	}
	defer f.Close(ctx)

	// Force buffered mode so the flush is the thing that hits the device error.
	if err := f.SetVBuf(ctx, "full", 4096); err != nil {
		t.Fatalf("setvbuf: %v", err)
	}

	// Buffered write succeeds (just fills the buffer).
	if err := f.Write(ctx, "hello"); err != nil {
		t.Fatalf("buffered write should succeed, got: %v", err)
	}

	// Flush must surface the device error (ENOSPC).
	if err := f.Flush(ctx); err == nil {
		t.Fatalf("expected flush error on /dev/full, got nil")
	}

	// After the reset, a small buffered write fits in freed buffer space and
	// must NOT re-return the sticky error. Without the reset, Go's bufio.Writer
	// retains the undeliverable bytes and re-returns the same error here.
	if err := f.Write(ctx, "x"); err != nil {
		t.Fatalf("post-flush buffered write should succeed (sticky error not cleared): %v", err)
	}
}
