# Print Provider Example

Shows how to route Lua `print()` and `warn()` output through a custom `vm.LuaPrintProvider`.

## Run

```bash
go run ./examples/print_provider
```

## Highlights

- prefix output for per-script logging
- collect prints and warns for testing or structured processing
- keep warn state isolated per VM
