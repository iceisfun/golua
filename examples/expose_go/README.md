# Exposing Go Functions to Lua

Demonstrates how to:
- Create native Go functions callable from Lua
- Access function arguments
- Return single or multiple values
- Create modules (tables of functions)
- Handle variadic arguments

## Run

```bash
go run ./examples/expose_go
```

## Output

```
Unix timestamp:	1234567890
Repeat:	GoGoGo
17 / 5 =	3	remainder	2
Point:	10	20
Upper:	HELLO
Lower:	world
Trim:	spaces
Sum:	15

All examples completed!
```

## Key Pattern

```go
v.SetGlobal("functionName", vm.NewNativeFunc(func(v *vm.VM) int {
    // Get arguments (1-indexed)
    arg1 := v.Get(1).AsString()
    arg2 := v.Get(2).AsInt()

    // Do work...
    result := doSomething(arg1, arg2)

    // Set return values (0-indexed)
    v.Set(0, vm.NewString(result))

    return 1 // number of return values
}))
```

## Creating Modules

```go
mylib := vm.NewEmptyTable()
mylib.SetString("func1", vm.NewNativeFunc(...))
mylib.SetString("func2", vm.NewNativeFunc(...))
v.SetGlobal("mylib", vm.NewTable(mylib))
```

Then in Lua: `mylib.func1()`, `mylib.func2()`
