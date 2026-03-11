# Timeout Example

Waits for a process with a timeout and kills it if it runs too long.

## Run

```bash
go run ./examples/exec/timeout
```

## Demonstrates

- `p:wait(ms)` timed waits
- explicit process termination with `p:kill()`
- reading the final exit result after termination

Requires the `sleep` command.
