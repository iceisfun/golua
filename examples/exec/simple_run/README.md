# exec.run Example

Runs short-lived commands and inspects their final result table.

## Run

```bash
go run ./examples/exec/simple_run
```

## Demonstrates

- basic command execution
- working directory and environment overrides
- merged stderr and per-call timeout options

Requires Unix-style commands like `ls` and `sh`.
