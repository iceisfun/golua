package vm

// LuaIoCaps declares which IO operations are allowed.
type LuaIoCaps struct {
	AllowRead  bool
	AllowWrite bool
}

// LuaIoProvider is a capability interface for IO operations.
// Implementations decide how and where files can be opened.
type LuaIoProvider interface {
	// Open opens a file with the given mode.
	// mode follows Lua conventions: "r", "w", "a", "rb", "wb", "ab"
	Open(name string, mode string) (LuaFile, error)

	// Capabilities declares which IO behaviors are allowed.
	Capabilities() LuaIoCaps
}

// LuaFile represents an open file handle.
type LuaFile interface {
	// Read reads data according to a format string.
	// Supported formats: "a" (all), "l" (line without newline), "L" (line with newline), "n" (number)
	Read(format string) (string, error)

	// ReadBytes reads up to n bytes from the file.
	ReadBytes(n int) (string, error)

	// Close closes the file.
	Close() error

	// IsClosed returns true if the file has been closed.
	IsClosed() bool
}
