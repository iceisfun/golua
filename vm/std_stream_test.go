package vm

// A provider instance is normally handed to several VMs, and each VM may run on
// its own goroutine, so the standard streams a provider exposes are shared. The
// buffered reader behind a standard stream must therefore tolerate concurrent
// readers: they may interleave line by line, but no read may observe a torn
// buffer. Reference Lua takes the same stream lock around its buffered reads
// (flockfile/funlockfile in liolib.c).

import (
	"context"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
)

// stdStreamLine is long enough that a torn buffer shows up as wrong content
// rather than only as a lost byte.
const stdStreamLine = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// pipeStdFile returns a readable stdFile fed by a goroutine writing n copies of
// stdStreamLine, plus a cleanup that closes the read end.
func pipeStdFile(t *testing.T, n int) *stdFile {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	t.Cleanup(func() { r.Close() })
	go func() {
		defer w.Close()
		for i := 0; i < n; i++ {
			if _, err := io.WriteString(w, stdStreamLine+"\n"); err != nil {
				return
			}
		}
	}()
	return &stdFile{file: r, name: "stdin", readable: true}
}

// TestStdFileConcurrentReadLines has several goroutines read lines from one
// shared standard stream, as several VMs sharing a provider do. Every line must
// come back whole and the stream must yield exactly the lines written.
func TestStdFileConcurrentReadLines(t *testing.T) {
	const lines = 4000
	const readers = 4

	f := pipeStdFile(t, lines)
	ctx := context.Background()

	var mu sync.Mutex
	total := 0
	var bad []string

	var wg sync.WaitGroup
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				line, err := f.Read(ctx, "l")
				if err != nil {
					return
				}
				mu.Lock()
				total++
				if line != stdStreamLine {
					bad = append(bad, line)
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if len(bad) > 0 {
		t.Errorf("%d of %d lines came back corrupted, first: %q", len(bad), total, bad[0])
	}
	if total != lines {
		t.Errorf("read %d lines, want %d", total, lines)
	}
}

// TestStdFileConcurrentReadBytes is the same check for counted reads, which
// drive the shared buffer through a different path.
func TestStdFileConcurrentReadBytes(t *testing.T) {
	const lines = 2000
	const readers = 4
	const chunk = len(stdStreamLine) + 1

	f := pipeStdFile(t, lines)
	ctx := context.Background()

	var mu sync.Mutex
	var got strings.Builder

	var wg sync.WaitGroup
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				s, err := f.ReadBytes(ctx, chunk)
				if err != nil {
					return
				}
				mu.Lock()
				got.WriteString(s)
				mu.Unlock()
				if len(s) < chunk {
					return
				}
			}
		}()
	}
	wg.Wait()

	want := strings.Repeat(stdStreamLine+"\n", lines)
	if got.Len() != len(want) {
		t.Fatalf("read %d bytes, want %d", got.Len(), len(want))
	}
	// Order across goroutines is not fixed, but each chunk must be one whole
	// record, so the whole stream must still be that record repeated.
	for i, s := 0, got.String(); i < len(s); i += chunk {
		if s[i:i+chunk] != stdStreamLine+"\n" {
			t.Fatalf("chunk at offset %d is torn: %q", i, s[i:i+chunk])
		}
	}
}
