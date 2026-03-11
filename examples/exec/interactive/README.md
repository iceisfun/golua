# Interactive Process Example

Starts a process, writes to stdin, then reads its output back from Lua.

## Run

```bash
go run ./examples/exec/interactive
```

## Demonstrates

- writing to a child process with `p:write()`
- closing stdin with `p:close_stdin()`
- reading sorted output line by line

Requires the `sort` command.
