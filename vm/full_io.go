package vm

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// FullIoProvider is a full-capability IO provider that allows both reading
// and writing files. It jails file access to a root directory for security.
type FullIoProvider struct {
	root   string
	stdin  *stdFile
	stdout *stdFile
	stderr *stdFile
}

// NewFullIoProvider creates a new FullIoProvider rooted at the given directory.
// All file operations are restricted to paths within this directory.
func NewFullIoProvider(root string) *FullIoProvider {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		absRoot = root
	}
	return &FullIoProvider{
		root:   absRoot,
		stdin:  &stdFile{file: os.Stdin, name: "stdin", readable: true},
		stdout: &stdFile{file: os.Stdout, name: "stdout", writable: true},
		stderr: &stdFile{file: os.Stderr, name: "stderr", writable: true},
	}
}

// resolvePath resolves a filename to an absolute path within the root.
func (p *FullIoProvider) resolvePath(name string) (string, error) {
	if filepath.IsAbs(name) {
		// Allow absolute paths that are within root
		absName, err := filepath.Abs(name)
		if err != nil {
			return "", err
		}
		// Check if path is within root or in temp directory
		if !strings.HasPrefix(absName, p.root) && !strings.HasPrefix(absName, os.TempDir()) {
			return "", fmt.Errorf("access denied: %s", name)
		}
		return absName, nil
	}
	return filepath.Join(p.root, name), nil
}

// Open opens a file within the provider's root directory.
func (p *FullIoProvider) Open(name string, mode string) (LuaFile, error) {
	// Reject empty filenames — Go's os.Open("") opens cwd as a directory,
	// but C's fopen("", ...) returns ENOENT which is what Lua expects.
	if name == "" {
		return nil, &os.PathError{Op: "open", Path: name, Err: syscall.ENOENT}
	}

	path, err := p.resolvePath(name)
	if err != nil {
		return nil, err
	}

	var flag int
	switch strings.TrimSuffix(mode, "b") {
	case "r":
		flag = os.O_RDONLY
	case "w":
		flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	case "a":
		flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	case "r+":
		flag = os.O_RDWR
	case "w+":
		flag = os.O_RDWR | os.O_CREATE | os.O_TRUNC
	case "a+":
		flag = os.O_RDWR | os.O_CREATE | os.O_APPEND
	default:
		return nil, fmt.Errorf("invalid mode: %s", mode)
	}

	f, err := os.OpenFile(path, flag, 0666)
	if err != nil {
		return nil, err
	}

	return &fullFile{
		file:   f,
		reader: bufio.NewReader(f),
		writer: bufio.NewWriter(f),
	}, nil
}

// Capabilities returns caps with full read/write access.
func (p *FullIoProvider) Capabilities() LuaIoCaps {
	return LuaIoCaps{
		AllowRead:  true,
		AllowWrite: true,
	}
}

// Stdin returns the standard input file handle.
func (p *FullIoProvider) Stdin() LuaFile { return p.stdin }

// Stdout returns the standard output file handle.
func (p *FullIoProvider) Stdout() LuaFile { return p.stdout }

// Stderr returns the standard error file handle.
func (p *FullIoProvider) Stderr() LuaFile { return p.stderr }

// TmpName creates a temporary file name. It creates a temp file, closes it,
// removes it, and returns the path -- matching Lua 5.4 os.tmpname behavior.
func (p *FullIoProvider) TmpName() (string, error) {
	f, err := os.CreateTemp("", "lua_")
	if err != nil {
		return "", err
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return name, nil
}

// Remove removes a file or empty directory.
func (p *FullIoProvider) Remove(name string) error {
	path, err := p.resolvePath(name)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

// TmpFile creates and opens a temporary file for read/write.
// The file is automatically removed when closed.
func (p *FullIoProvider) TmpFile() (LuaFile, error) {
	f, err := os.CreateTemp("", "lua_tmpfile_")
	if err != nil {
		return nil, err
	}
	// Remove the file immediately so it's cleaned up when closed
	os.Remove(f.Name())
	return &fullFile{
		file:   f,
		reader: bufio.NewReader(f),
		writer: bufio.NewWriter(f),
	}, nil
}

// Rename renames (moves) a file within the provider's root directory.
func (p *FullIoProvider) Rename(oldname, newname string) error {
	oldpath, err := p.resolvePath(oldname)
	if err != nil {
		return err
	}
	newpath, err := p.resolvePath(newname)
	if err != nil {
		return err
	}
	return os.Rename(oldpath, newpath)
}

// fullFile wraps an os.File with buffered reading and writing.
type fullFile struct {
	file    *os.File
	reader  *bufio.Reader
	writer  *bufio.Writer
	closed  bool
	bufMode string // "no", "full", "line"
	bufSize int    // buffer size for "full" mode
}

func (f *fullFile) Read(format string) (string, error) {
	if f.closed {
		return "", fmt.Errorf("attempt to use a closed file")
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
			return f.readLine(false)

		case 'L':
			return f.readLine(true)

		case 'n':
			return f.readNumber()
		}
	}

	return "", fmt.Errorf("invalid read format: %s", format)
}

func (f *fullFile) ReadBytes(n int) (string, error) {
	if f.closed {
		return "", fmt.Errorf("attempt to use a closed file")
	}
	if n < 0 {
		return "", fmt.Errorf("not enough memory")
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

func (f *fullFile) Write(data string) error {
	if f.closed {
		return fmt.Errorf("attempt to use a closed file")
	}

	if f.bufMode == "no" || f.bufMode == "" {
		// Unbuffered: write directly
		_, err := f.file.Write([]byte(data))
		return err
	}

	// Buffered: write through buffer
	_, err := f.writer.WriteString(data)
	if err != nil {
		return err
	}

	if f.bufMode == "line" {
		// Line buffered: flush if data contains newline
		if strings.ContainsRune(data, '\n') {
			return f.writer.Flush()
		}
	}
	return nil
}

func (f *fullFile) Seek(whence string, offset int64) (int64, error) {
	if f.closed {
		return 0, fmt.Errorf("attempt to use a closed file")
	}

	// Flush any buffered writes before seeking
	if f.writer != nil {
		f.writer.Flush()
	}

	var w int
	switch whence {
	case "set":
		w = io.SeekStart
	case "cur":
		w = io.SeekCurrent
	case "end":
		w = io.SeekEnd
	default:
		return 0, fmt.Errorf("invalid option '%s'", whence)
	}

	// When seeking relative to "cur", the OS file descriptor may be ahead
	// of the logical position due to bufio.Reader read-ahead buffering.
	// Adjust the offset to compensate for unconsumed buffered bytes.
	if w == io.SeekCurrent && f.reader != nil {
		buffered := f.reader.Buffered()
		if buffered > 0 {
			offset -= int64(buffered)
		}
	}

	pos, err := f.file.Seek(offset, w)
	if err != nil {
		return 0, err
	}

	// Reset reader after seek so it doesn't return stale buffered data
	f.reader.Reset(f.file)

	return pos, nil
}

func (f *fullFile) Flush() error {
	if f.closed {
		return fmt.Errorf("attempt to use a closed file")
	}
	if f.writer != nil {
		return f.writer.Flush()
	}
	return nil
}

func (f *fullFile) SetVBuf(mode string, size int) error {
	if f.closed {
		return fmt.Errorf("attempt to use a closed file")
	}

	switch mode {
	case "no":
		// Flush any existing buffer
		if f.writer != nil {
			f.writer.Flush()
		}
		f.bufMode = "no"
		f.writer = nil
	case "full":
		if f.writer != nil {
			f.writer.Flush()
		}
		f.bufMode = "full"
		f.bufSize = size
		if size > 0 {
			f.writer = bufio.NewWriterSize(f.file, size)
		} else {
			f.writer = bufio.NewWriter(f.file)
		}
	case "line":
		if f.writer != nil {
			f.writer.Flush()
		}
		f.bufMode = "line"
		if size > 0 {
			f.writer = bufio.NewWriterSize(f.file, size)
		} else {
			f.writer = bufio.NewWriter(f.file)
		}
	default:
		return fmt.Errorf("invalid option '%s'", mode)
	}
	return nil
}

func (f *fullFile) Close() error {
	if f.closed {
		return fmt.Errorf("cannot close a closed file")
	}
	f.closed = true
	// Flush buffer before closing
	if f.writer != nil {
		f.writer.Flush()
	}
	return f.file.Close()
}

func (f *fullFile) IsClosed() bool {
	return f.closed
}

func (f *fullFile) IsStd() bool {
	return false
}

// readLine reads a line from the buffered reader.
func (f *fullFile) readLine(keepNewline bool) (string, error) {
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
func (f *fullFile) readNumber() (string, error) {
	if f.writer != nil {
		f.writer.Flush()
	}

	return readNumberFromBuf(f.reader)
}

// readNumberFromBuf reads a number from a bufio.Reader.
// It skips leading whitespace (which is not restored on failure),
// reads number-like characters, and finds the longest valid prefix
// that parses as a number. Uses Peek to avoid consuming bytes that
// aren't part of the number.
func readNumberFromBuf(reader *bufio.Reader) (string, error) {
	// Skip leading whitespace
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return "", err
		}
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			reader.UnreadByte()
			break
		}
	}

	// State-machine scanner matching Lua 5.4's read_number (liolib.c).
	// On failure, all scanned characters are consumed (not restored).
	const maxNumLen = 200 // Lua 5.4's L_MAXLENNUM

	offset := 0
	tooLong := false

	peekByte := func() (byte, bool) {
		if offset >= maxNumLen {
			tooLong = true
			return 0, false
		}
		peeked, _ := reader.Peek(offset + 1)
		if len(peeked) <= offset {
			return 0, false
		}
		return peeked[offset], true
	}

	accept := func() { offset++ }

	isDigit := func(b byte) bool { return b >= '0' && b <= '9' }
	isHexDigit := func(b byte) bool {
		return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
	}

	fail := func() (string, error) {
		if offset > 0 {
			reader.Discard(offset)
		}
		return "", fmt.Errorf("not a number")
	}

	// Optional sign
	if b, ok := peekByte(); ok && (b == '+' || b == '-') {
		accept()
	}

	// Determine if hex (0x/0X prefix)
	isHex := false
	hasDigits := false
	if b, ok := peekByte(); ok && b == '0' {
		accept()
		hasDigits = true // the '0' itself is a valid digit
		if b2, ok2 := peekByte(); ok2 && (b2 == 'x' || b2 == 'X') {
			accept()
			isHex = true
			hasDigits = false // need at least one hex digit after 0x
		}
	}

	digitCheck := isDigit
	if isHex {
		digitCheck = isHexDigit
	}

	// Read integer part digits
	for {
		b, ok := peekByte()
		if !ok || !digitCheck(b) {
			break
		}
		accept()
		hasDigits = true
	}

	// Optional fractional part
	if b, ok := peekByte(); ok && b == '.' {
		accept()
		for {
			b, ok := peekByte()
			if !ok || !digitCheck(b) {
				break
			}
			accept()
			hasDigits = true
		}
	}

	// Must have at least some digits
	if !hasDigits {
		return fail()
	}

	// Optional exponent (e/E for decimal, p/P for hex)
	expChar := byte('e')
	expCharU := byte('E')
	if isHex {
		expChar = 'p'
		expCharU = 'P'
	}
	if b, ok := peekByte(); ok && (b == expChar || b == expCharU) {
		accept()
		// Optional exponent sign
		if b2, ok2 := peekByte(); ok2 && (b2 == '+' || b2 == '-') {
			accept()
		}
		// Exponent digits (always decimal, even for hex floats)
		hasExpDigits := false
		for {
			b, ok := peekByte()
			if !ok || !isDigit(b) {
				break
			}
			accept()
			hasExpDigits = true
		}
		if !hasExpDigits {
			// Incomplete exponent is a hard failure in Lua 5.4
			return fail()
		}
	}

	// If the number exceeded the maximum length, it's invalid.
	if tooLong {
		return fail()
	}

	// Extract the scanned string and consume it
	peeked, _ := reader.Peek(offset)
	result := string(peeked[:offset])
	reader.Discard(offset)
	return result, nil
}

// stdFile wraps an os.File for standard input/output/error.
// It cannot be closed by the user.
type stdFile struct {
	file     *os.File
	name     string
	readable bool
	writable bool
	reader   *bufio.Reader
}

func (f *stdFile) ensureReader() *bufio.Reader {
	if f.reader == nil {
		f.reader = bufio.NewReader(f.file)
	}
	return f.reader
}

func (f *stdFile) Read(format string) (string, error) {
	if !f.readable {
		return "", fmt.Errorf("%s is not readable", f.name)
	}
	r := f.ensureReader()
	// Normalize format: strip leading * and check first character
	cleanFmt := strings.TrimPrefix(format, "*")
	if len(cleanFmt) > 0 {
		switch cleanFmt[0] {
		case 'a':
			data, err := io.ReadAll(r)
			if err != nil {
				return "", err
			}
			return string(data), nil
		case 'l':
			return f.readLine(r, false)
		case 'L':
			return f.readLine(r, true)
		case 'n':
			return readNumberFromBuf(r)
		}
	}
	return "", fmt.Errorf("invalid read format: %s", format)
}

func (f *stdFile) ReadBytes(n int) (string, error) {
	if !f.readable {
		return "", fmt.Errorf("%s is not readable", f.name)
	}
	r := f.ensureReader()
	if n == 0 {
		// EOF test: peek 1 byte to check if data remains
		_, err := r.Peek(1)
		if err != nil {
			return "", io.EOF
		}
		return "", nil
	}
	buf := make([]byte, n)
	nRead, err := io.ReadFull(r, buf)
	if nRead == 0 && err != nil {
		return "", err
	}
	return string(buf[:nRead]), nil
}

func (f *stdFile) Write(data string) error {
	if !f.writable {
		return fmt.Errorf("%s is not writable", f.name)
	}
	_, err := f.file.Write([]byte(data))
	return err
}

func (f *stdFile) Seek(whence string, offset int64) (int64, error) {
	return 0, fmt.Errorf("cannot seek on %s", f.name)
}

func (f *stdFile) Flush() error {
	if !f.writable {
		return fmt.Errorf("%s is not writable", f.name)
	}
	return nil
}

func (f *stdFile) SetVBuf(mode string, size int) error {
	return nil // No-op for std files
}

func (f *stdFile) Close() error {
	return fmt.Errorf("cannot close standard file")
}

func (f *stdFile) IsClosed() bool {
	return false
}

func (f *stdFile) IsStd() bool {
	return true
}

func (f *stdFile) readLine(r *bufio.Reader, keepNewline bool) (string, error) {
	var line strings.Builder
	for {
		b, err := r.ReadByte()
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
