// Example: Lua 5.5 language features
//
// This example demonstrates several Lua 5.5 features beyond global declarations:
// - Named vararg parameters: function f(... args)
// - Prefix attribute syntax: local<const> x = 1
// - table.create(narr, nrec) for preallocated tables
// - For-loop read-only control variables (compile-time error)
// - error(nil) producing "<no error object>"
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
	// 1. Named vararg parameters
	// ---------------------------------------------------------------
	fmt.Println("=== 1. Named vararg parameters ===")
	err := run(`
		-- In Lua 5.5, "... name" collects varargs into a table with an .n field.
		-- The ... expression still works alongside the named table.

		local function show(... args)
			print("  args.n =", args.n)
			for i = 1, args.n do
				print("  args[" .. i .. "] =", args[i])
			end
		end

		print("show(10, 20, 30):")
		show(10, 20, 30)

		print("show() with no arguments:")
		show()
	`)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	// ---------------------------------------------------------------
	// 2. Named vararg with regular parameters
	// ---------------------------------------------------------------
	fmt.Println("\n=== 2. Named vararg with regular parameters ===")
	err = run(`
		-- Regular parameters come first, then ... name collects the rest
		local function log(level, ... messages)
			local parts = {}
			for i = 1, messages.n do
				parts[i] = tostring(messages[i])
			end
			print("[" .. level .. "] " .. table.concat(parts, " "))
		end

		log("INFO", "Server started on port", 8080)
		log("WARN", "Memory usage:", 85, "percent")
		log("ERROR", "Connection failed")
	`)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	// ---------------------------------------------------------------
	// 3. Named vararg preserves nil arguments
	// ---------------------------------------------------------------
	fmt.Println("\n=== 3. Named vararg preserves nil arguments ===")
	err = run(`
		-- The .n field counts all arguments including nils,
		-- unlike #table which stops at the first nil
		local function count_args(... args)
			print("  args.n =", args.n)
			print("  #args  =", #args)
			for i = 1, args.n do
				print("  args[" .. i .. "] =", tostring(args[i]))
			end
		end

		print("count_args(1, nil, 3, nil):")
		count_args(1, nil, 3, nil)
	`)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	// ---------------------------------------------------------------
	// 4. Named vararg: ... expression still works
	// ---------------------------------------------------------------
	fmt.Println("\n=== 4. Named vararg: ... expression coexists ===")
	err = run(`
		-- Both the named table and the ... expression are available
		local function forward(... args)
			-- Use ... to forward to another function
			print("via ...:", ...)
			-- Use the named table for indexed access
			print("via args:", args[1], args[2])
		end

		forward("hello", "world")
	`)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	// ---------------------------------------------------------------
	// 5. Named vararg is read-only (const)
	// ---------------------------------------------------------------
	fmt.Println("\n=== 5. Named vararg parameter is const ===")
	err = run(`
		-- The named vararg parameter itself is const -- you cannot reassign
		-- the variable, though you can modify the table's contents.
		local ok, msg = load("return function(... t) t = 10 end")
		print("  Compile error:", ok == nil)
		print("  Message:", msg)
	`)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	// ---------------------------------------------------------------
	// 6. Prefix attribute syntax: local<const>
	// ---------------------------------------------------------------
	fmt.Println("\n=== 6. Prefix attribute syntax ===")
	err = run(`
		-- Lua 5.5 allows the <const> attribute before the variable name
		-- as a prefix, in addition to the Lua 5.4 suffix syntax.

		-- Prefix syntax: local<const> name = value
		local<const> MAX = 100
		local<const> GREETING = "Hello"
		print("  MAX =", MAX, "  GREETING =", GREETING)

		-- Suffix syntax still works: local name <const> = value
		local LIMIT <const> = 50
		print("  LIMIT =", LIMIT)

		-- Prefix applies to all names in the declaration
		local<const> A, B, C = 1, 2, 3
		print("  A =", A, "  B =", B, "  C =", C)
	`)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	// Demonstrate the compile-time error for assigning to a prefix-const
	fmt.Println("Attempting to assign to local<const>...")
	err = run(`
		local<const> X = 1
		X = 2
	`)
	if err != nil {
		fmt.Printf("  Caught expected error: %v\n", err)
	} else {
		log.Fatal("Expected a compile error but got none")
	}

	// ---------------------------------------------------------------
	// 7. table.create(narr, nrec)
	// ---------------------------------------------------------------
	fmt.Println("\n=== 7. table.create(narr, nrec) ===")
	err = run(`
		-- table.create preallocates a table with capacity hints.
		-- The returned table is always empty.

		-- Create a table with space for 5 array elements and 2 hash entries
		local t = table.create(5, 2)
		print("  type =", type(t), "  #t =", #t)

		-- Fill it up -- no rehashing needed
		for i = 1, 5 do
			t[i] = i * 10
		end
		t.name = "example"
		t.active = true

		print("  After filling:")
		print("  #t =", #t)
		print("  t[1] =", t[1], "  t[5] =", t[5])
		print("  t.name =", t.name, "  t.active =", t.active)

		-- No-arg version: same as {}
		local empty = table.create()
		print("  table.create() type =", type(empty), "  #empty =", #empty)
	`)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	// ---------------------------------------------------------------
	// 8. For-loop control variables are read-only
	// ---------------------------------------------------------------
	fmt.Println("\n=== 8. For-loop read-only control variables ===")

	// Normal for-loop usage works fine
	err = run(`
		local sum = 0
		for i = 1, 5 do
			sum = sum + i
		end
		print("  Sum 1..5 =", sum)
	`)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	// Assigning to numeric for control variable is a compile-time error
	fmt.Println("Attempting to assign to numeric for variable...")
	err = run(`
		for i = 1, 10 do
			i = 5
		end
	`)
	if err != nil {
		fmt.Printf("  Caught expected error: %v\n", err)
	} else {
		log.Fatal("Expected a compile error but got none")
	}

	// Assigning to generic for control variable (first variable) is an error
	fmt.Println("Attempting to assign to generic for control variable...")
	err = run(`
		for k, v in pairs({a=1}) do
			k = "modified"
		end
	`)
	if err != nil {
		fmt.Printf("  Caught expected error: %v\n", err)
	} else {
		log.Fatal("Expected a compile error but got none")
	}

	// Second variable in generic for is NOT read-only
	err = run(`
		for k, v in pairs({a=1, b=2}) do
			v = v * 10  -- this is fine
		end
		print("  Assigning to non-control variable: OK")
	`)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	// Shadowing with local is fine
	err = run(`
		local results = {}
		for i = 1, 3 do
			local i = i + 10  -- shadowing is allowed
			results[#results + 1] = i
		end
		print("  Shadowed values:", table.concat(results, ", "))
	`)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	// ---------------------------------------------------------------
	// 9. error(nil) produces "<no error object>"
	// ---------------------------------------------------------------
	fmt.Println("\n=== 9. error(nil) produces '<no error object>' ===")
	err = run(`
		-- In Lua 5.5, error(nil) replaces nil with a descriptive string

		-- error(nil)
		local ok, msg = pcall(error, nil)
		print("  error(nil):", type(msg), msg)

		-- error() with no arguments
		local ok2, msg2 = pcall(error)
		print("  error():   ", type(msg2), msg2)

		-- error(false) is NOT replaced (only nil is special)
		local ok3, msg3 = pcall(error, false)
		print("  error(false):", type(msg3), msg3)

		-- Regular string errors work as before
		local ok4, msg4 = pcall(error, "something went wrong")
		print("  error(str): ", type(msg4), msg4)

		-- xpcall: handler sees nil, but final result is replaced if nil
		local ok5, msg5 = xpcall(
			function() error(nil) end,
			function(e)
				print("  Handler received:", type(e), e)
				return nil  -- returning nil from handler also gets replaced
			end
		)
		print("  xpcall result:", type(msg5), msg5)

		-- assert(nil, nil) also gets the replacement
		local ok6, msg6 = pcall(assert, nil, nil)
		print("  assert(nil,nil):", type(msg6), msg6)
	`)
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}

	fmt.Println("\nAll Lua 5.5 feature examples completed!")
}
