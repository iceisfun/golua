package vm

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"
)

// JailedIoProvider is a read-only, filesystem-jailed IO provider.
// It restricts file access to a specific directory using os.DirFS.
type JailedIoProvider struct {
	fs fs.FS
}

// NewJailedIoProvider creates a new JailedIoProvider rooted at the given directory.
// All file operations are restricted to paths within this directory.
func NewJailedIoProvider(root string) *JailedIoProvider {
	return &JailedIoProvider{
		fs: os.DirFS(root),
	}
}

// Open opens a file within the jailed directory. Only read modes ("r", "rb")
// are permitted; write attempts return an error.
func (p *JailedIoProvider) Open(name string, mode string) (LuaFile, error) {
	// Only allow read modes
	if mode != "r" && mode != "rb" {
		return nil, fmt.Errorf("JailedIoProvider: write mode '%s' not permitted", mode)
	}

	f, err := p.fs.Open(name)
	if err != nil {
		return nil, err
	}

	return &jailedFile{
		file:   f,
		reader: bufio.NewReader(f),
	}, nil
}

// Capabilities returns caps with read-only access enabled.
func (p *JailedIoProvider) Capabilities() LuaIoCaps {
	return LuaIoCaps{
		AllowRead:  true,
		AllowWrite: false,
	}
}

// Stdin returns nil (not supported in jailed provider).
func (p *JailedIoProvider) Stdin() LuaFile { return nil }

// Stdout returns nil (not supported in jailed provider).
func (p *JailedIoProvider) Stdout() LuaFile { return nil }

// Stderr returns nil (not supported in jailed provider).
func (p *JailedIoProvider) Stderr() LuaFile { return nil }

// TmpName is not supported in jailed provider.
func (p *JailedIoProvider) TmpName() (string, error) {
	return "", fmt.Errorf("os.tmpname not available in jailed IO provider")
}

// Remove is not supported in jailed provider.
func (p *JailedIoProvider) Remove(name string) error {
	return fmt.Errorf("os.remove not available in jailed IO provider")
}

// TmpFile is not supported in jailed provider.
func (p *JailedIoProvider) TmpFile() (LuaFile, error) {
	return nil, fmt.Errorf("io.tmpfile not available in jailed IO provider")
}

// Rename is not supported in jailed provider.
func (p *JailedIoProvider) Rename(oldname, newname string) error {
	return fmt.Errorf("os.rename not available in jailed IO provider")
}

// jailedFile wraps an fs.File with buffered reading.
type jailedFile struct {
	file   fs.File
	reader *bufio.Reader
	closed bool
}

func (f *jailedFile) Read(format string) (string, error) {
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

func (f *jailedFile) ReadBytes(n int) (string, error) {
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

	buf := make([]byte, n)
	nRead, err := io.ReadFull(f.reader, buf)
	if nRead == 0 && err != nil {
		return "", err
	}
	return string(buf[:nRead]), nil
}

func (f *jailedFile) Close() error {
	if f.closed {
		return fmt.Errorf("attempt to close a closed file")
	}
	f.closed = true
	return f.file.Close()
}

func (f *jailedFile) IsClosed() bool {
	return f.closed
}

func (f *jailedFile) Write(data string) error {
	return fmt.Errorf("write not supported in jailed IO provider")
}

func (f *jailedFile) Seek(whence string, offset int64) (int64, error) {
	return 0, fmt.Errorf("seek not supported in jailed IO provider")
}

func (f *jailedFile) Flush() error {
	return fmt.Errorf("flush not supported in jailed IO provider")
}

func (f *jailedFile) SetVBuf(mode string, size int) error {
	return fmt.Errorf("setvbuf not supported in jailed IO provider")
}

func (f *jailedFile) IsStd() bool {
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
// On failure, the file position is restored if the underlying file supports seeking.
func (f *jailedFile) readNumber() (string, error) {
	// Try to save position for restore on failure
	var startPos int64
	var canSeek bool
	if seeker, ok := f.file.(io.Seeker); ok {
		buffered := f.reader.Buffered()
		rawPos, err := seeker.Seek(0, io.SeekCurrent)
		if err == nil {
			startPos = rawPos - int64(buffered)
			canSeek = true
		}
	}

	result, parseErr := f.tryReadNumber()
	if parseErr != nil && canSeek {
		if seeker, ok := f.file.(io.Seeker); ok {
			seeker.Seek(startPos, io.SeekStart)
			f.reader.Reset(f.file)
		}
	}
	if parseErr != nil {
		return "", parseErr
	}
	return result, nil
}

// tryReadNumber attempts to read a number token from the stream.
func (f *jailedFile) tryReadNumber() (string, error) {
	// Skip whitespace
	for {
		b, err := f.reader.ReadByte()
		if err != nil {
			return "", err
		}
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			f.reader.UnreadByte()
			break
		}
	}

	// Read number characters
	var num strings.Builder
	for {
		b, err := f.reader.ReadByte()
		if err != nil {
			if err == io.EOF && num.Len() > 0 {
				break
			}
			return "", err
		}
		if (b >= '0' && b <= '9') || b == '.' || b == '-' || b == '+' || b == 'e' || b == 'E' || b == 'x' || b == 'X' ||
			(b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F') {
			num.WriteByte(b)
		} else {
			f.reader.UnreadByte()
			break
		}
	}

	s := num.String()
	if _, err := strconv.ParseFloat(s, 64); err != nil {
		if _, err := strconv.ParseInt(s, 0, 64); err != nil {
			return "", fmt.Errorf("not a number: %s", s)
		}
	}
	return s, nil
}
