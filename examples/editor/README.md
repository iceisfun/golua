# GoLua Editor

Browser-based Lua editor with live diagnostics and execution.

## Run

```bash
go run ./examples/editor/
# Open http://127.0.0.1:8080
```

## Features

- Monaco editor with Lua syntax highlighting
- Live diagnostics (red squiggles) as you type
- Run button (or Ctrl+Enter) executes Lua in a sandboxed VM
- Output panel shows print() results and errors

## Security

The VM is sandboxed: 5s timeout, 10M instruction limit, no file/OS/network access.
