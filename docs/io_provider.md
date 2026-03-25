# IO Provider

The `LuaIoProvider` interface controls all `io.*` file operations, `os.remove`, `os.rename`, and `os.tmpname`. Without an IO provider, the `io` table is absent.

## Quick Start

```go
// Read-only, directory-confined
v := vm.New()
if err := v.SetIoProvider(vm.NewJailedIoProvider("/app/data")); err != nil {
    log.Fatal(err)
}
stdlib.Open(v)

// Full read-write access
v := vm.New()
if err := v.SetIoProvider(vm.NewFullIoProvider("/app/data")); err != nil {
    log.Fatal(err)
}
stdlib.Open(v)
```

## Interface

```go
type LuaIoProvider interface {
    Open(ctx context.Context, name string, mode string) (LuaFile, error)
    Capabilities(ctx context.Context) LuaIoCaps
    Stdin(ctx context.Context) LuaFile
    Stdout(ctx context.Context) LuaFile
    Stderr(ctx context.Context) LuaFile
    TmpName(ctx context.Context) (string, error)
    Remove(ctx context.Context, name string) error
    Rename(ctx context.Context, oldname, newname string) error
    TmpFile(ctx context.Context) (LuaFile, error)
}

type LuaIoCaps struct {
    AllowRead  bool
    AllowWrite bool
}

type LuaFile interface {
    Read(ctx context.Context, format string) (string, error)
    ReadBytes(ctx context.Context, n int) (string, error)
    Write(ctx context.Context, data string) error
    Seek(ctx context.Context, whence string, offset int64) (int64, error)
    Flush(ctx context.Context) error
    SetVBuf(ctx context.Context, mode string, size int) error
    Close(ctx context.Context) error
    IsClosed(ctx context.Context) bool
    IsStd(ctx context.Context) bool
}
```

### Capabilities

- `AllowRead` — enables `io.open` for reading, `io.lines`, `io.read`
- `AllowWrite` — enables `io.open` for writing, `io.write`, `io.tmpfile`

### LuaFile

File objects support Lua 5.4 read formats (`"a"`, `"l"`, `"L"`, `"n"`), byte reads, writing, seeking (`"set"`, `"cur"`, `"end"`), flushing, and buffering modes (`"no"`, `"full"`, `"line"`).

`IsStd` returns true for stdin/stdout/stderr handles (which cannot be closed by Lua code). `IsClosed` tracks whether `Close` has been called.

## Default Implementations

### JailedIoProvider

```go
provider := vm.NewJailedIoProvider("/app/data")
```

Read-only filesystem access confined to a single directory:

- Only `"r"` and `"rb"` modes allowed; write modes return an error
- Capabilities: `AllowRead=true, AllowWrite=false`
- `Stdin`/`Stdout`/`Stderr` return nil (no stdio)
- `TmpName`, `Remove`, `Rename`, `TmpFile` return errors
- Path traversal outside the root is prevented

### FullIoProvider

```go
provider := vm.NewFullIoProvider("/app/data")
```

Full read-write access confined to a root directory (plus the OS temp directory):

- All standard modes: `"r"`, `"w"`, `"a"`, `"rb"`, `"wb"`, `"ab"`, `"r+"`, `"w+"`, `"a+"`
- Capabilities: `AllowRead=true, AllowWrite=true`
- `Stdin`/`Stdout`/`Stderr` return wrapped OS file handles
- `TmpName` creates and removes a temp file, returning the path
- `TmpFile` creates a temp file that is auto-removed on close
- `Remove` deletes files and empty directories
- `Rename` moves files within the jailed path
- Buffering modes supported: unbuffered, fully buffered, line-buffered

## Custom Implementation

Implement `LuaIoProvider` and `LuaFile` to back `io.*` with any storage — embedded filesystems, database blobs, in-memory buffers, or network mounts.

```go
type EmbedIoProvider struct{ fsys fs.FS }

func (p *EmbedIoProvider) Open(ctx context.Context, name, mode string) (vm.LuaFile, error) { /* ... */ }
func (p *EmbedIoProvider) Capabilities(ctx context.Context) vm.LuaIoCaps {
    return vm.LuaIoCaps{AllowRead: true, AllowWrite: false}
}
func (p *EmbedIoProvider) Stdin(ctx context.Context) vm.LuaFile  { return nil }
func (p *EmbedIoProvider) Stdout(ctx context.Context) vm.LuaFile { return nil }
func (p *EmbedIoProvider) Stderr(ctx context.Context) vm.LuaFile { return nil }
func (p *EmbedIoProvider) TmpName(ctx context.Context) (string, error)           { return "", fmt.Errorf("unsupported") }
func (p *EmbedIoProvider) TmpFile(ctx context.Context) (vm.LuaFile, error)       { return nil, fmt.Errorf("unsupported") }
func (p *EmbedIoProvider) Remove(ctx context.Context, _ string) error            { return fmt.Errorf("unsupported") }
func (p *EmbedIoProvider) Rename(ctx context.Context, _, _ string) error         { return fmt.Errorf("unsupported") }
```

## Security

- Without a provider, `io.*` does not exist
- `JailedIoProvider` enforces read-only access and directory confinement
- `FullIoProvider` confines writes to the root directory and OS temp directory
- No stdio by default in library mode (the CLI provides stdio via its own environment)
