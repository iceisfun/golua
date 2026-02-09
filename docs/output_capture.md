# Output Capture for Testing

GoLua supports capturing `print()` output for testing, available both from Go and Lua.

## Go API

### Enable Capture

Use `vm.WithCaptureOutput(true)` when creating a VM:

```go
v := vm.New(vm.WithCaptureOutput(true))
stdlib.Open(v)

v.Run(proto)

// Retrieve captured output
lines := v.OutputLines()    // []string of all printed lines
last := v.LastOutput()      // most recent line, or ""
v.ClearOutput()             // reset the buffer
```

### Example: Testing Print Output

```go
func TestMyScript(t *testing.T) {
    v := vm.New(vm.WithCaptureOutput(true))
    stdlib.Open(v)

    proto, _ := compile(`
        print("hello", "world")
        print(1 + 2)
    `)
    v.Run(proto)

    lines := v.OutputLines()
    assert(lines[0] == "hello\tworld")
    assert(lines[1] == "3")
}
```

### Coroutine Support

Output capture is automatically inherited by coroutine VMs. Print calls from
inside coroutines are captured in the same buffer as the parent VM.

## Lua API

Two functions are available for inspecting captured output from within Lua:

```lua
-- Get the most recent print() output line
local line = _lastoutput()

-- Get all captured lines as a table
local lines = _outputlines()
for i, line in ipairs(lines) do
    print(i, line)  -- note: this also appends to the buffer
end
```

These are primarily useful for self-checking tests when capture is enabled.
When capture is disabled, `_lastoutput()` returns `""` and `_outputlines()`
returns an empty table.

## Doctest Harness

The test framework includes a doctest harness for validating print output
against inline `-->` directives in Lua source files.

### Directive Format

```lua
print("hello")
--> =hello              -- exact match

print(type(42))
--> =number              -- exact match

print(tostring({}))
--> ~^table: 0x          -- regex match (prefix ~)

local ok, err = pcall(error, "boom")
print(ok, err)
--> ~false\t.*boom       -- regex: false + tab + anything + boom
```

- `-->  =text` — exact string match against the output line
- `-->  ~pattern` — Go regexp match against the output line
- Directives are matched against `print()` output lines in order
- Lines without `-->` produce output that is not checked

### Running Proposed Tests

Proposed test files live in `proposed_tests/` and run via:

```bash
go test ./tests/ -run TestProposed -v
```

The harness reports each directive's pass/fail status, making it easy to see
which behaviors are implemented and which are still missing.
