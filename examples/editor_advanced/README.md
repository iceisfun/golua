# Advanced Editor Example

Serves a browser-based Lua IDE with live diagnostics, completion, hover, and sandboxed execution.

## Run

```bash
go run ./examples/editor_advanced
# Open http://127.0.0.1:8080
```

## Features

- Monaco editor UI with JSON-RPC 2.0 language services
- live diagnostics from the `check` package, mapped to Monaco markers
- scope-aware completion (locals, parameters, for-variables, functions, globals)
  plus curated stdlib metadata in `language/stdlib.go`
- hover docs for stdlib symbols and user-declared names
- sandboxed execution with a wall-clock timeout, instruction/stack limits, and
  no host providers (no filesystem, environment, or process access)

## How it stays accurate

- **Metadata drift guard** — `language/stdlib.go` is hand-written prose, but its
  *set* of symbols is verified against a live VM by `TestStdlibMetadataMatchesVM`
  in `language/stdlib_drift_test.go`. If a stdlib change adds or removes a
  function, the test fails and points at the metadata to update. Run:

  ```bash
  go test ./examples/editor_advanced/...
  ```

- **Span-based scopes** — `language/scope.go` derives scope boundaries from the
  parser's AST `End()` spans rather than guessing from sibling positions, so
  completion correctly stops offering a local/parameter once its block ends.

## Notes

- completion and hover are intentionally lightweight and example-sized, not a
  full LSP implementation
- the Run sandbox closes its VM after each request (`vm.Close`) to release
  providers, run close hooks, and discard pending finalizers
- the frontend loads Monaco from a CDN, so first load requires internet access
