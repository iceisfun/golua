# Code Provider

The `LuaCodeProvider` interface controls what Lua scripts can load via `dofile()`, `loadfile()`, and `require`. Without a code provider, these functions are unavailable.

## Quick Start

```go
v := vm.New()
if err := v.SetCodeProvider(vm.NewDirCodeProvider("/app/scripts", vm.LuaLoaderCaps{
    AllowDofile:   true,
    AllowLoadfile: true,
})); err != nil {
    log.Fatal(err)
}
stdlib.Open(v)
```

## Interface

```go
type LuaCodeProvider interface {
    LoadChunk(ctx context.Context, name string, caller *LuaCallerContext) (source []byte, chunkName string, err error)
    Capabilities(ctx context.Context) LuaLoaderCaps
}

type LuaLoaderCaps struct {
    AllowDofile   bool
    AllowLoadfile bool
}

type LuaCallerContext struct {
    ScriptName string
    VMID       string
    CallDepth  int
}
```

### LoadChunk

Resolves a chunk name to source code. Returns the source bytes, a display name for error messages, and an error if the chunk cannot be found.

The `caller` argument provides the requesting script's name, VM ID, and call depth for audit logging or access control.

### Capabilities

Controls which loading functions are registered when `stdlib.Open` runs:

- `AllowDofile` — registers `dofile()`
- `AllowLoadfile` — registers `loadfile()`

`load()` is always available (loads from strings, not files).

The `require` module searcher uses `LoadChunk` to find Lua modules. If no code provider is set, only `package.preload` and `package.loaded` searchers are active.

## Default Implementation: DirCodeProvider

```go
provider := vm.NewDirCodeProvider("/app/scripts", vm.LuaLoaderCaps{
    AllowDofile:   true,
    AllowLoadfile: true,
})
```

`DirCodeProvider` loads files from a root directory using `os.DirFS`. Paths are cleaned and confined to the root directory (plus the OS temp directory for compatibility). Path traversal outside the root is rejected.

## Custom Implementation

```go
type InMemoryLoader struct {
    files map[string]string
}

func (l *InMemoryLoader) LoadChunk(ctx context.Context, name string, caller *vm.LuaCallerContext) ([]byte, string, error) {
    src, ok := l.files[name]
    if !ok {
        return nil, "", fmt.Errorf("module %q not found", name)
    }
    return []byte(src), "@" + name, nil
}

func (l *InMemoryLoader) Capabilities(ctx context.Context) vm.LuaLoaderCaps {
    return vm.LuaLoaderCaps{AllowDofile: true, AllowLoadfile: true}
}
```

## Security

- Without a provider, `dofile` and `loadfile` do not exist
- Capabilities independently gate each function
- `DirCodeProvider` confines access to a single directory tree
- Custom providers can implement any access policy (allowlists, audit logging, rate limiting)
