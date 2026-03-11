# Advanced Editor Example

Serves a browser-based Lua IDE with live diagnostics, completion, hover, and sandboxed execution.

## Run

```bash
go run ./examples/editor_advanced
# Open http://127.0.0.1:8080
```

## Features

- Monaco editor UI with JSON-RPC 2.0 language services
- live diagnostics from the `check` package
- completion and hover from curated stdlib metadata in `examples/editor_advanced/language/stdlib.go`
- sandboxed execution with a 5s timeout and instruction limits

## Notes

- completion and hover are intentionally lightweight and example-sized, not a full LSP implementation
- the frontend loads Monaco from a CDN, so first load requires internet access
