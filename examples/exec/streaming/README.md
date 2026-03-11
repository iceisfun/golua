# exec.spawn Streaming Example

Streams stdout line by line from a running process.

## Run

```bash
go run ./examples/exec/streaming
```

## Demonstrates

- incremental reads with `p:readlines()`
- waiting for process completion
- merging stderr into stdout for a single output stream

Requires a Unix-style shell with `sh`.
