# Check Example

Demonstrates the `check` package: parses incomplete Lua source and prints the
partial AST alongside JSON diagnostics suitable for the Monaco editor.

Unlike the reference parser — which stops at the first syntax error — `check`
performs **multi-error recovery**, so a single call can surface several
independent problems. Each diagnostic also carries a stable, machine-readable
**`code`** (e.g. `unexpected-symbol`, `unfinished-string`, `token-expected`)
that editor tooling can use for filtering or quick-fix routing, independent of
the human-readable message. The conformance-critical parser is never modified;
recovery happens entirely in the `check` layer.

## Run

```bash
go run ./examples/check/
```
