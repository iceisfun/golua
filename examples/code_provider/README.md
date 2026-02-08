# Sandboxed Code Loading with LuaCodeProvider

Demonstrates how to control what Lua code can be loaded at runtime using `LuaCodeProvider`.

## Use Cases

- **Embedded scripts**: Store Lua scripts in your Go binary
- **Database-backed**: Load scripts from a database
- **Virtual filesystem**: Map script names to different sources
- **Security sandbox**: Restrict which scripts can be loaded
- **Audit logging**: Track what code is being loaded and by whom

## Run

```bash
go run main.go
```

## Output

```
=== Running with InMemoryProvider ===

[Provider] Loading 'utils.lua' (requested by: main.lua)
[Provider] Loading 'config.lua' (requested by: main.lua)
Config version:	1.0.0
Debug mode:	true
Double 21:	42
Greeting:	Hello, World
[Provider] Loading 'utils.lua' (requested by: main.lua)
Loaded again:	20

=== Running with RestrictedProvider ===

[Provider] Loading 'utils.lua' (requested by: restricted.lua)
Utils loaded:	10
Expected error:	access denied: config.lua

=== Complete ===
```

## The LuaCodeProvider Interface

```go
type LuaCodeProvider interface {
    // LoadChunk resolves and returns Lua source code
    LoadChunk(name string, caller *LuaCallerContext) (source []byte, chunkName string, err error)

    // Capabilities declares what's allowed
    Capabilities() LuaLoaderCaps
}

type LuaCallerContext struct {
    ScriptName string // which script is requesting
    VMID       string // optional VM identifier
    CallDepth  int    // call stack depth
}

type LuaLoaderCaps struct {
    AllowDofile   bool // enable dofile()
    AllowLoadfile bool // enable loadfile()
}
```

## Common Patterns

### In-Memory Scripts

```go
provider := NewInMemoryProvider()
provider.Add("module.lua", `return { foo = 42 }`)
v.SetCodeProvider(provider)
```

### Filesystem with Restrictions

```go
type SafeFileProvider struct {
    basePath string
    allowed  []string
}

func (p *SafeFileProvider) LoadChunk(name string, caller *LuaCallerContext) ([]byte, string, error) {
    // Validate path is within allowed directories
    fullPath := filepath.Join(p.basePath, name)
    if !isWithinAllowed(fullPath, p.allowed) {
        return nil, "", errors.New("access denied")
    }
    source, err := os.ReadFile(fullPath)
    return source, "@" + name, err
}
```

### Logging/Auditing

```go
func (p *AuditProvider) LoadChunk(name string, caller *LuaCallerContext) ([]byte, string, error) {
    log.Printf("AUDIT: %s loading %s at depth %d",
        caller.ScriptName, name, caller.CallDepth)
    return p.inner.LoadChunk(name, caller)
}
```
