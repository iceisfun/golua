// Example: Streaming process output with exec.spawn.
//
// Demonstrates spawning a process and reading its output line by line,
// including merging stderr into stdout with {merge_stderr = true}.
package main

import (
	"fmt"
	"log"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

func main() {
	fmt.Println("=== Streaming Example ===")
	fmt.Println()

	source := `
		-- Stream stdout line by line
		local p = exec.spawn("sh", "-c", "for i in 1 2 3 4 5; do echo line_$i; done")

		for line in p:readlines() do
			print("Got:", line)
		end

		local result = p:wait()
		print("Exit code:", result.code)

		-- Merge stderr into stdout for unified streaming
		print("\n--- Merged output ---")
		local p2 = exec.spawn("sh", "-c", "echo stdout_line; echo stderr_line >&2", {merge_stderr = true})
		for line in p2:readlines() do
			print("Merged:", line)
		end
		p2:wait()
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
