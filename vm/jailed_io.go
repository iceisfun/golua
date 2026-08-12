package vm

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"syscall"
)

// JailedIoProvider is a read-only, filesystem-jailed IO provider.
// It restricts file access to a specific directory.
type JailedIoProvider struct {
	jail *pathJail
}

// NewJailedIoProvider creates a new JailedIoProvider rooted at the given directory.
// All file operations are restricted to paths within this directory: a name is
// resolved, symlinks included, and refused unless it lands inside the root.
// os.DirFS is not enough on its own — it rejects "../x" but happily follows a
// symlink inside the root to anywhere the link points.
//
// An empty root admits nothing; a provider that reads from everywhere is asked
// for explicitly, with AllowRoot.
func NewJailedIoProvider(root string) *JailedIoProvider {
	return &JailedIoProvider{
		jail: newPathJail(root),
	}
}

// AllowRoot widens the jail with another readable directory. See
// FullIoProvider.AllowRoot; call it before the provider is handed to a VM.
func (p *JailedIoProvider) AllowRoot(dir string) { p.jail.allowRoot(dir) }

// Open opens a file within the jailed directory. Only read modes ("r", "rb")
// are permitted; write attempts return an error.
func (p *JailedIoProvider) Open(ctx context.Context, name string, mode string) (LuaFile, error) {
	// Only allow read modes
	if mode != "r" && mode != "rb" {
		return nil, fmt.Errorf("JailedIoProvider: write mode '%s' not permitted", mode)
	}
	// Go's os.Open("") opens the cwd as a directory; C's fopen("") is ENOENT,
	// which is what Lua reports.
	if name == "" {
		return nil, &fs.PathError{Op: "open", Path: name, Err: syscall.ENOENT}
	}

	path, err := p.jail.resolve(name)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return &jailedFile{
		file:   f,
		reader: bufio.NewReader(f),
	}, nil
}

// Capabilities returns caps with read-only access enabled.
func (p *JailedIoProvider) Capabilities(ctx context.Context) LuaIoCaps {
	return LuaIoCaps{
		AllowRead:  true,
		AllowWrite: false,
	}
}

// Stdin returns nil (not supported in jailed provider).
func (p *JailedIoProvider) Stdin(ctx context.Context) LuaFile { return nil }

// Stdout returns nil (not supported in jailed provider).
func (p *JailedIoProvider) Stdout(ctx context.Context) LuaFile { return nil }

// Stderr returns nil (not supported in jailed provider).
func (p *JailedIoProvider) Stderr(ctx context.Context) LuaFile { return nil }

// TmpName is not supported in jailed provider.
func (p *JailedIoProvider) TmpName(ctx context.Context) (string, error) {
	return "", fmt.Errorf("os.tmpname not available in jailed IO provider")
}

// Remove is not supported in jailed provider.
func (p *JailedIoProvider) Remove(ctx context.Context, name string) error {
	return fmt.Errorf("os.remove not available in jailed IO provider")
}

// TmpFile is not supported in jailed provider.
func (p *JailedIoProvider) TmpFile(ctx context.Context) (LuaFile, error) {
	return nil, fmt.Errorf("io.tmpfile not available in jailed IO provider")
}

// Rename is not supported in jailed provider.
func (p *JailedIoProvider) Rename(ctx context.Context, oldname, newname string) error {
	return fmt.Errorf("os.rename not available in jailed IO provider")
}

// jailedFile wraps an fs.File with buffered reading.
type jailedFile struct {
	file   fs.File
	reader *bufio.Reader
	closed bool
}

func (f *jailedFile) Read(ctx context.Context, format string) (string, error) {
	if f.closed {
		return "", fmt.Errorf("attempt to read from a closed file")
	}

	// Normalize format: strip leading * and check first character
	// This matches Lua 5.4 behavior where "all" == "a", "line" == "l", etc.
	cleanFmt := strings.TrimPrefix(format, "*")
	if len(cleanFmt) > 0 {
		switch cleanFmt[0] {
		case 'a':
			data, err := io.ReadAll(f.reader)
			if err != nil {
				return "", err
			}
			return string(data), nil

		case 'l':
			line, err := f.readLine(false)
			if err != nil {
				return "", err
			}
			return line, nil

		case 'L':
			line, err := f.readLine(true)
			if err != nil {
				return "", err
			}
			return line, nil

		case 'n':
			return f.readNumber()
		}
	}

	return "", fmt.Errorf("invalid read format: %s", format)
}

func (f *jailedFile) ReadBytes(ctx context.Context, n int) (string, error) {
	if f.closed {
		return "", fmt.Errorf("attempt to read from a closed file")
	}
	if n == 0 {
		// EOF test: peek 1 byte to check if data remains
		_, err := f.reader.Peek(1)
		if err != nil {
			return "", io.EOF
		}
		return "", nil
	}
	if n < 0 {
		return "", fmt.Errorf("not enough memory")
	}

	// Grow with the data that arrives rather than allocating n up front: a
	// script names the count for free, and make([]byte, 1<<30) on a two-byte
	// file is a Go runtime OOM no pcall can catch.
	return readAtMost(f.reader, n, availableFor(n, f.available))
}

// available estimates the bytes still readable, so ReadBytes can size its
// buffer from the file rather than from the count. Zero means "unknown".
func (f *jailedFile) available() int {
	info, err := f.file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return 0
	}
	// fs.File carries no position, so the whole size is the only bound
	// available; it is still the size of something that exists.
	return clampToInt(info.Size())
}

func (f *jailedFile) Close(ctx context.Context) error {
	if f.closed {
		return fmt.Errorf("attempt to close a closed file")
	}
	f.closed = true
	return f.file.Close()
}

func (f *jailedFile) IsClosed(ctx context.Context) bool {
	return f.closed
}

func (f *jailedFile) Write(ctx context.Context, data string) error {
	return fmt.Errorf("write not supported in jailed IO provider")
}

func (f *jailedFile) Seek(ctx context.Context, whence string, offset int64) (int64, error) {
	return 0, fmt.Errorf("seek not supported in jailed IO provider")
}

func (f *jailedFile) Flush(ctx context.Context) error {
	return fmt.Errorf("flush not supported in jailed IO provider")
}

func (f *jailedFile) SetVBuf(ctx context.Context, mode string, size int) error {
	return fmt.Errorf("setvbuf not supported in jailed IO provider")
}

func (f *jailedFile) IsStd(ctx context.Context) bool {
	return false
}

// readLine reads a line from the buffered reader.
// If keepNewline is true, the trailing newline is included.
func (f *jailedFile) readLine(keepNewline bool) (string, error) {
	var line strings.Builder
	for {
		b, err := f.reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				if line.Len() > 0 {
					return line.String(), nil
				}
				return "", io.EOF
			}
			return "", err
		}
		if b == '\n' {
			if keepNewline {
				line.WriteByte('\n')
			}
			return line.String(), nil
		}
		line.WriteByte(b)
	}
}

// readNumber reads and parses a number from the stream, skipping leading whitespace.
// Consumed whitespace is not restored on failure (matches Lua 5.4 behavior).
func (f *jailedFile) readNumber() (string, error) {
	return readNumberFromBuf(f.reader)
}
