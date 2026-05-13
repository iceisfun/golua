# check — Lua Diagnostics for Editor Integration

The `check` package parses partial/incomplete Lua source and returns diagnostics
with positions matching Monaco's 1-based `IMarkerData` interface.

## Usage

```go
import "github.com/iceisfun/golua/v1/check"

result := check.Check("editor.lua", sourceText)

// result.Block is never nil — it contains the partial AST
// (everything that parsed successfully before the first error).
for _, d := range result.Diagnostics {
    fmt.Printf("Line %d, Col %d: %s\n",
        d.StartLineNumber, d.StartColumn, d.Message)
}
```

## Monaco Integration

Diagnostics map directly to Monaco's `IMarkerData`:

```typescript
const markers = diagnostics.map(d => ({
    severity: d.severity,        // matches monaco.MarkerSeverity
    message:  d.message,
    startLineNumber: d.startLineNumber,
    startColumn:     d.startColumn,
    endLineNumber:   d.endLineNumber,
    endColumn:       d.endColumn,
}));
monaco.editor.setModelMarkers(model, "lua", markers);
```

## Severity Constants

| Constant       | Value | Monaco equivalent            |
|----------------|-------|------------------------------|
| `check.Hint`   | 1     | `monaco.MarkerSeverity.Hint` |
| `check.Info`   | 2     | `monaco.MarkerSeverity.Info` |
| `check.Warning`| 4     | `monaco.MarkerSeverity.Warning` |
| `check.Error`  | 8     | `monaco.MarkerSeverity.Error` |

## API

### `Check(source, input string) *Result`

Parses `input` as Lua source and returns a `Result`.

- `source` — name used in error messages (e.g. a filename)
- `input` — the Lua source text

### `Result`

```go
type Result struct {
    Block       *ast.Block   // partial AST, never nil
    Diagnostics []Diagnostic // list of diagnostics
}

func (r *Result) HasErrors() bool
```

### `Diagnostic`

```go
type Diagnostic struct {
    Severity        int    // Hint, Info, Warning, or Error
    Message         string // human-readable error message
    StartLineNumber int    // 1-based line
    StartColumn     int    // 1-based column
    EndLineNumber   int    // 1-based line
    EndColumn       int    // 1-based column
}
```

## Limitations

- **One error per call** — the parser stops at the first error (no recovery).
- **End = Start** — the parser doesn't track token spans, so end positions equal
  start positions.
- **Error severity only** — only `Error` severity is currently produced (no
  warnings or hints yet).
