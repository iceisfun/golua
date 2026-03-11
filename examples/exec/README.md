# Exec Examples

Examples for the optional `exec` module powered by `vm.LuaProcessProvider`.

## Run

```bash
go run ./examples/exec/simple_run
go run ./examples/exec/streaming
go run ./examples/exec/interactive
go run ./examples/exec/timeout
```

## Notes

- these examples use common Unix commands such as `sh`, `ls`, `sort`, and `sleep`
- they are best run on Linux or macOS, or in a Unix-like shell environment
- each example enables the module with `v.SetProcessProvider(vm.NewDefaultProcessProvider())`
