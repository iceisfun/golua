// Example: Exposing Go functions to Lua
//
// This example shows how to:
// - Create native Go functions callable from Lua
// - Access function arguments
// - Return values to Lua
// - Create modules (tables of functions)
package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

func main() {
	v := vm.New()
	stdlib.Open(v)

	// Example 1: Simple function with no arguments
	v.SetGlobal("now", vm.NewNativeFunc(func(v *vm.VM) int {
		v.Set(0, vm.NewInt(time.Now().Unix()))
		return 1 // return 1 value
	}))

	// Example 2: Function with arguments
	v.SetGlobal("repeat_string", vm.NewNativeFunc(func(v *vm.VM) int {
		s := v.Get(1).AsString()
		n := int(v.Get(2).AsInt())
		v.Set(0, vm.NewString(strings.Repeat(s, n)))
		return 1
	}))

	// Example 3: Function with multiple return values
	v.SetGlobal("divmod", vm.NewNativeFunc(func(v *vm.VM) int {
		a := v.Get(1).AsInt()
		b := v.Get(2).AsInt()
		v.Set(0, vm.NewInt(a/b)) // quotient
		v.Set(1, vm.NewInt(a%b)) // remainder
		return 2 // return 2 values
	}))

	// Example 4: Function that returns a table
	v.SetGlobal("make_point", vm.NewNativeFunc(func(v *vm.VM) int {
		x := v.Get(1)
		y := v.Get(2)
		t := vm.NewEmptyTable()
		t.SetString("x", x)
		t.SetString("y", y)
		v.Set(0, vm.NewTable(t))
		return 1
	}))

	// Example 5: Create a module (table of functions)
	mylib := vm.NewEmptyTable()

	mylib.SetString("upper", vm.NewNativeFunc(func(v *vm.VM) int {
		s := v.Get(1).AsString()
		v.Set(0, vm.NewString(strings.ToUpper(s)))
		return 1
	}))

	mylib.SetString("lower", vm.NewNativeFunc(func(v *vm.VM) int {
		s := v.Get(1).AsString()
		v.Set(0, vm.NewString(strings.ToLower(s)))
		return 1
	}))

	mylib.SetString("trim", vm.NewNativeFunc(func(v *vm.VM) int {
		s := v.Get(1).AsString()
		v.Set(0, vm.NewString(strings.TrimSpace(s)))
		return 1
	}))

	v.SetGlobal("mylib", vm.NewTable(mylib))

	// Example 6: Variadic function (access all arguments)
	v.SetGlobal("sum_all", vm.NewNativeFunc(func(v *vm.VM) int {
		n := v.ArgCount()
		var sum int64 = 0
		for i := 1; i <= n; i++ {
			sum += v.Get(i).AsInt()
		}
		v.Set(0, vm.NewInt(sum))
		return 1
	}))

	// Lua code that uses our Go functions
	source := `
		-- Use simple function
		print("Unix timestamp:", now())

		-- Use function with arguments
		print("Repeat:", repeat_string("Go", 3))

		-- Use multiple return values
		local q, r = divmod(17, 5)
		print("17 / 5 =", q, "remainder", r)

		-- Use function returning table
		local p = make_point(10, 20)
		print("Point:", p.x, p.y)

		-- Use module
		print("Upper:", mylib.upper("hello"))
		print("Lower:", mylib.lower("WORLD"))
		print("Trim:", mylib.trim("  spaces  "))

		-- Use variadic function
		print("Sum:", sum_all(1, 2, 3, 4, 5))
	`

	block, err := parser.Parse("expose_go", source)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	proto, err := compiler.Compile("expose_go", block)
	if err != nil {
		log.Fatalf("Compile error: %v", err)
	}

	_, err = v.Run(proto)
	if err != nil {
		log.Fatalf("Runtime error: %v", err)
	}

	fmt.Println("\nAll examples completed!")
}
