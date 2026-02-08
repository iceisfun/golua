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

func (p *JailedIoProvider) Capabilities() LuaIoCaps {
	return LuaIoCaps{
		AllowRead:  true,
		AllowWrite: false,
	}
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

	switch format {
	case "a", "*a":
		// Read entire file
		data, err := io.ReadAll(f.reader)
		if err != nil {
			return "", err
		}
		return string(data), nil

	case "l", "*l":
		// Read line without trailing newline
		line, err := f.readLine(false)
		if err != nil {
			return "", err
		}
		return line, nil

	case "L", "*L":
		// Read line with trailing newline
		line, err := f.readLine(true)
		if err != nil {
			return "", err
		}
		return line, nil

	case "n", "*n":
		// Read a number
		return f.readNumber()

	default:
		return "", fmt.Errorf("invalid read format: %s", format)
	}
}

func (f *jailedFile) ReadBytes(n int) (string, error) {
	if f.closed {
		return "", fmt.Errorf("attempt to read from a closed file")
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
func (f *jailedFile) readNumber() (string, error) {
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
