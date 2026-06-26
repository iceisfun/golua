# GoLua Editor

A browser-based Lua editor with live diagnostics and sandboxed execution,
built on the [Monaco editor](https://microsoft.github.io/monaco-editor/) and
golua's `check` package.

## Run

```bash
go run ./examples/editor/
# Open http://127.0.0.1:8080
```

## Features

- Monaco editor with Lua syntax highlighting
- Live diagnostics (red squiggles) as you type, via `POST /api/check`
- Run button (or Ctrl/Cmd+Enter) executes Lua in a fresh sandboxed VM
- Output panel shows `print()` results, run duration, and errors
- Runtime errors are marked inline on the offending line and revealed in view
- Status bar summarizes diagnostics ("No issues" / error & warning counts)

## How it works

- **`POST /api/check`** calls `check.Check`, which parses partial/incomplete
  source and returns `check.Diagnostic` values. Their fields already match
  Monaco's `IMarkerData` (1-based positions, severity = `MarkerSeverity`), so
  they map straight to `monaco.editor.setModelMarkers`.
- **`POST /api/run`** parses, compiles, and runs the source in a brand-new VM
  per request. It returns captured output plus a structured error: the message
  is phase-tagged (`parse error:` / `compile error:` / `runtime error:`) and
  `errorLine` carries the 1-based line so the frontend can place a marker.

## Security / sandbox

Each run gets a fresh VM that is sandboxed two ways:

- **By limits** (`vm.Limits`): a 5s wall-clock deadline (`vm.WithContext`), a
  50M-instruction budget, plus call-depth and stack-slot caps. The deadline is
  derived from the HTTP request context, so a client disconnect also cancels
  the script.
- **By omission**: no `io` / `os` / `exec` / network providers are registered,
  so the script gets only pure computation and the safe standard library
  (`string`, `table`, `math`, …). It cannot touch the filesystem, environment,
  or network. Request bodies are capped at 256 KiB.

`v.Run` is additionally wrapped in `recover()` so a single bad script can never
crash the server.

## See also

`examples/editor_advanced/` adds JSON-RPC language services (completion, hover)
on top of the same execution model.
