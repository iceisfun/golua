// Example: Simple process execution with exec.run.
//
// Demonstrates running a command with options: cwd, env, merge_stderr, timeout.
package main

import (
	"fmt"
	"log"

	"github.com/iceisfun/golua/v1/compiler"
	"github.com/iceisfun/golua/v1/parser"
	"github.com/iceisfun/golua/v1/stdlib"
	"github.com/iceisfun/golua/v1/vm"
)

func main() {
	fmt.Println("=== exec.run Example ===")
	fmt.Println()

	source := `
		-- Basic run
		local r = exec.run("ls", "-al")
		print("Exit code:", r.code)
		print("Success:", r.success)
		print("Output:")
		print(r.stdout)

		-- Run with options
		print("\n--- With options ---")
		local r2 = exec.run("ls", {
			cwd = "/tmp",
			merge_stderr = true,
			timeout = 5000
		})
		print("Listing /tmp:")
		print(r2.stdout)

		-- Custom environment
		local r3 = exec.run("sh", "-c", "echo Hello from $USER_NAME", {
			env = {USER_NAME = "GoLua"}
		})
		print(r3.stdout)
	`

	block, err := parser.Parse("example", source)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	proto, err := compiler.Compile("example", block)
	if err != nil {
		log.Fatalf("Compile error: %v", err)
	}

	v := vm.New()
	if err := v.SetProcessProvider(vm.NewDefaultProcessProvider()); err != nil {
		log.Fatalf("SetProcessProvider: %v", err)
	}
	stdlib.Open(v)

	_, err = v.Run(proto)
	if err != nil {
		log.Fatalf("Runtime error: %v", err)
	}
}
