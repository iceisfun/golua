package vm

// LuaIoCaps declares which IO operations are allowed.
type LuaIoCaps struct {
	AllowRead  bool
	AllowWrite bool
}

// LuaIoProvider is a capability interface for sandboxed file I/O.
// When provided to a VM, the stdlib io library becomes available.
// Without a provider, io.* functions are not registered.
//
// Lua 5.4 Reference: §6.8 (input and output facilities).
type LuaIoProvider interface {
	// Open opens a file with the given mode.
	// mode follows Lua conventions: "r", "w", "a", "rb", "wb", "ab"
	Open(name string, mode string) (LuaFile, error)

	// Capabilities declares which IO behaviors are allowed.
	Capabilities() LuaIoCaps

	// Stdin returns a file handle for standard input.
	// Returns nil if not supported.
	Stdin() LuaFile

	// Stdout returns a file handle for standard output.
	// Returns nil if not supported.
	Stdout() LuaFile

	// Stderr returns a file handle for standard error.
	// Returns nil if not supported.
	Stderr() LuaFile

	// TmpName returns a unique temporary file name.
	// Returns empty string and error if not supported.
	TmpName() (string, error)

	// Remove removes a file or empty directory.
	// Returns error if not supported or if the operation fails.
	Remove(name string) error

	// Rename renames (moves) a file.
	// Returns error if not supported or if the operation fails.
	Rename(oldname, newname string) error

	// TmpFile creates and opens a temporary file for read/write.
	// The file is automatically removed when closed.
	// Returns error if not supported.
	TmpFile() (LuaFile, error)
}

// LuaFile represents an open file handle.
type LuaFile interface {
	// Read reads data according to a format string.
	// Supported formats: "a" (all), "l" (line without newline), "L" (line with newline), "n" (number)
	Read(format string) (string, error)

	// ReadBytes reads up to n bytes from the file.
	ReadBytes(n int) (string, error)

	// Write writes data to the file.
	Write(data string) error

	// Seek sets the file position. whence is "set", "cur", or "end".
	// Returns the new absolute position.
	Seek(whence string, offset int64) (int64, error)

	// Flush flushes any buffered data to the underlying file.
	Flush() error

	// SetVBuf sets the buffering mode. mode is "no", "full", or "line".
	// size is the buffer size (0 means default).
	SetVBuf(mode string, size int) error

	// Close closes the file.
	Close() error

	// IsClosed returns true if the file has been closed.
	IsClosed() bool

	// IsStd reports whether this is a standard file (stdin/stdout/stderr)
	// that should not be closable by the user.
	IsStd() bool
}

