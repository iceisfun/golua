package vm

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"testing"
)

// A read that has to publish pending writes first must report a failed flush
// rather than drop it: the bytes never reached the file, and bufio keeps the
// error sticky, so swallowing it only moves the failure to an unrelated later
// write or close.
func TestFullFileReadReportsFlushError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(path, []byte("data\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// A read-only descriptor makes the flush fail with EBADF.
	fd, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer fd.Close()
	f := &fullFile{
		file:     fd,
		reader:   bufio.NewReader(fd),
		writer:   bufio.NewWriter(fd),
		bufMode:  "full",
		readable: true,
		writable: true,
	}
	if _, err := f.writer.WriteString("x"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if s, err := f.Read(ctx, "l"); err == nil {
		t.Fatalf("Read after failed flush = %q, nil; want the flush error", s)
	}
	if s, err := f.ReadBytes(ctx, 1); err == nil {
		t.Fatalf("ReadBytes after failed flush = %q, nil; want the flush error", s)
	}
}
