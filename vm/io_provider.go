package vm

import "context"

// LuaIoCaps declares which io-library file operations are exposed to Lua.
type LuaIoCaps struct {
	// AllowRead enables read-oriented operations.
	AllowRead bool
	// AllowWrite enables write-oriented operations.
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
	Open(ctx context.Context, name string, mode string) (LuaFile, error)

	// Capabilities declares which IO behaviors are allowed.
	Capabilities(ctx context.Context) LuaIoCaps

	// Stdin returns a file handle for standard input.
	// Returns nil if not supported.
	Stdin(ctx context.Context) LuaFile

	// Stdout returns a file handle for standard output.
	// Returns nil if not supported.
	Stdout(ctx context.Context) LuaFile

	// Stderr returns a file handle for standard error.
	// Returns nil if not supported.
	Stderr(ctx context.Context) LuaFile

	// TmpName returns a unique temporary file name.
	// Returns empty string and error if not supported.
	TmpName(ctx context.Context) (string, error)

	// Remove removes a file or empty directory.
	// Returns error if not supported or if the operation fails.
	Remove(ctx context.Context, name string) error

	// Rename renames (moves) a file.
	// Returns error if not supported or if the operation fails.
	Rename(ctx context.Context, oldname, newname string) error

	// TmpFile creates and opens a temporary file for read/write.
	// The file is automatically removed when closed.
	// Returns error if not supported.
	TmpFile(ctx context.Context) (LuaFile, error)
}

// LuaFile represents an open file handle.
type LuaFile interface {
	// Read reads data according to a format string.
	// Supported formats: "a" (all), "l" (line without newline), "L" (line with newline), "n" (number)
	Read(ctx context.Context, format string) (string, error)

	// ReadBytes reads up to n bytes from the file.
	ReadBytes(ctx context.Context, n int) (string, error)

	// Write writes data to the file.
	Write(ctx context.Context, data string) error

	// Seek sets the file position. whence is "set", "cur", or "end".
	// Returns the new absolute position.
	Seek(ctx context.Context, whence string, offset int64) (int64, error)

	// Flush flushes any buffered data to the underlying file.
	Flush(ctx context.Context) error

	// SetVBuf sets the buffering mode. mode is "no", "full", or "line".
	// size is the buffer size (0 means default).
	SetVBuf(ctx context.Context, mode string, size int) error

	// Close closes the file.
	Close(ctx context.Context) error

	// IsClosed returns true if the file has been closed.
	IsClosed(ctx context.Context) bool

	// IsStd reports whether this is a standard file (stdin/stdout/stderr)
	// that should not be closable by the user.
	IsStd(ctx context.Context) bool
}
