# Capture Output Example

Shows how to enable `vm.WithCaptureOutput(true)` so `print()` output is stored in memory instead of written to stdout.

## Run

```bash
go run ./examples/capture_output
```

## Highlights

- inspect all captured lines with `v.OutputLines()`
- read the most recent line with `v.LastOutput()`
- reset the buffer with `v.ClearOutput()`
