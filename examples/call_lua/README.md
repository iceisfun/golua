# Calling Lua Functions from Go

Demonstrates how to:
- Define functions in Lua
- Call those functions from Go with arguments
- Handle multiple return values
- Pass and receive tables

## Run

```bash
go run ./examples/call_lua
```

## Output

```
greet("World") = Hello, World!
add(10, 20) = 30
getMultiple() = 1, 2, 3
process({x=5, y=3}) = {sum=8, product=15}
```

## Key APIs

```go
// Get a global Lua function
fn := v.GetGlobal("functionName")

// Call with arguments (protected from panics)
results, err := v.ProtectedCall(fn, []vm.Value{
    vm.NewInt(42),
    vm.NewString("hello"),
})

// Access results
results[0].AsInt()
results[0].AsString()
results[0].AsTable()
```
