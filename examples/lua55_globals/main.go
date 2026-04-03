// Example: Lua 5.5 global declarations
//
// This example demonstrates Lua 5.5's new global declaration system:
// - "global x, y" to explicitly declare globals
// - "global<const> PI = 3.14159" for constant globals
// - "global *" wildcard to allow all globals
// - Compile-time error detection for undeclared globals
// - "global" as a soft keyword (usable as a variable name)
package main

import (
	"fmt"
	"log"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

// helper compiles and runs Lua source, returning any error.
func run(source string) error {
	block, err := parser.Parse("example", source)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}
	proto, err := compiler.Compile("example", block)
	if err != nil {
		return fmt.Errorf("compile error: %w", err)
	}
	v := vm.New()
	stdlib.Open(v)
	_, err = v.Run(proto)
	return err
}

func main() {
	// ---------------------------------------------------------------
	// 1. Basic explicit global declarations
	// ---------------------------------------------------------------
	fmt.Println("=== 1. Explicit global declarations ===")
	err := run(`
		global x, y
		global print

		x = 10
		y = 20
		print("x =", x, "  y =", y, "  x + y =", x + y)
	`)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	// ---------------------------------------------------------------
	// 2. Constant globals with global<const>
	// ---------------------------------------------------------------
	fmt.Println("\n=== 2. Constant globals (global<const>) ===")
	err = run(`
		global print
		global<const> PI = 3.14159
		global<const> VERSION = "1.0.0"

		print("PI =", PI)
		print("VERSION =", VERSION)
	`)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	// ---------------------------------------------------------------
	// 3. Wildcard: global * allows all globals
	// ---------------------------------------------------------------
	fmt.Println("\n=== 3. Wildcard (global *) ===")
	err = run(`
		global *

		-- With global *, any name can be used freely
		message = "Hello from global *"
		print(message)

		counter = 0
		for i = 1, 5 do
			counter = counter + i
		end
		print("Sum 1..5 =", counter)
	`)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	// ---------------------------------------------------------------
	// 4. Compile-time error: undeclared global read
	// ---------------------------------------------------------------
	fmt.Println("\n=== 4. Compile-time error: undeclared global read ===")
	err = run(`
		global x
		global print
		-- Using 'y' without declaring it triggers a compile error
		print(y)
	`)
	if err != nil {
		fmt.Printf("Caught expected error: %v\n", err)
	} else {
		log.Fatal("Expected a compile error but got none")
	}

	// ---------------------------------------------------------------
	// 5. Compile-time error: undeclared global write
	// ---------------------------------------------------------------
	fmt.Println("\n=== 5. Compile-time error: undeclared global write ===")
	err = run(`
		global x
		-- Writing to 'z' without declaring it triggers a compile error
		z = 42
	`)
	if err != nil {
		fmt.Printf("Caught expected error: %v\n", err)
	} else {
		log.Fatal("Expected a compile error but got none")
	}

	// ---------------------------------------------------------------
	// 6. Compile-time error: assigning to a const global
	// ---------------------------------------------------------------
	fmt.Println("\n=== 6. Compile-time error: assigning to const global ===")
	err = run(`
		global<const> MAX = 100
		MAX = 200
	`)
	if err != nil {
		fmt.Printf("Caught expected error: %v\n", err)
	} else {
		log.Fatal("Expected a compile error but got none")
	}

	// ---------------------------------------------------------------
	// 7. Const wildcard: read OK, write error
	// ---------------------------------------------------------------
	fmt.Println("\n=== 7. Const wildcard (global<const> *) ===")
	err = run(`
		global<const> *
		-- Reading from globals is fine
		print("Read OK:", type(print))
	`)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	fmt.Println("Attempting write with const wildcard...")
	err = run(`
		global<const> *
		x = 1
	`)
	if err != nil {
		fmt.Printf("Caught expected error: %v\n", err)
	} else {
		log.Fatal("Expected a compile error but got none")
	}

	// ---------------------------------------------------------------
	// 8. Specific name overrides const wildcard
	// ---------------------------------------------------------------
	fmt.Println("\n=== 8. Specific declaration overrides const wildcard ===")
	err = run(`
		global counter
		global print
		global<const> *

		-- 'counter' was explicitly declared as read-write, so assignment works
		-- even though global<const> * is active
		counter = 0
		counter = counter + 1
		print("counter =", counter)
	`)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	// ---------------------------------------------------------------
	// 9. Global function declaration
	// ---------------------------------------------------------------
	fmt.Println("\n=== 9. Global function declaration ===")
	err = run(`
		global function greet(name)
			return "Hello, " .. name .. "!"
		end
		global print
		print(greet("Lua 5.5"))
	`)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	// ---------------------------------------------------------------
	// 10. Scoping: inner scope inherits globals, can add its own
	// ---------------------------------------------------------------
	fmt.Println("\n=== 10. Scope inheritance ===")
	err = run(`
		global x
		global print
		x = 100
		do
			global y
			y = 200
			print("Inner scope: x =", x, " y =", y)
		end
		print("Outer scope: x =", x)
	`)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	// ---------------------------------------------------------------
	// 11. "global" is a soft keyword
	// ---------------------------------------------------------------
	fmt.Println("\n=== 11. Soft keyword: 'global' as a variable name ===")

	// "global" used as a local variable name
	err = run(`
		local global = 42
		print(global)
	`)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	// "global" used as a table field
	err = run(`
		local t = { global = "field value" }
		print(t.global)
	`)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	// "global = expr" at statement start is a plain assignment, not a declaration
	err = run(`
		global = 99
		print(global)
	`)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	// "global" as a function parameter name
	err = run(`
		local function f(global)
			return global * 2
		end
		print(f(21))
	`)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	// ---------------------------------------------------------------
	// 12. Integration: const initializer + rw globals + functions
	// ---------------------------------------------------------------
	fmt.Println("\n=== 12. Integration example ===")
	err = run(`
		global print, type, tostring
		global count = 0
		global<const> VERSION = "5.5"
		global<const> MAX_RETRIES = 3

		global function increment()
			count = count + 1
		end

		for i = 1, MAX_RETRIES do
			increment()
		end

		print("Version " .. VERSION .. ": count = " .. tostring(count))
		print("MAX_RETRIES =", MAX_RETRIES, " type =", type(MAX_RETRIES))
	`)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	fmt.Println("\nAll global declaration examples completed!")
}
