// Example: Streaming process output with exec.spawn.
//
// Demonstrates spawning a process and reading its output line by line.
package main

import (
	"fmt"
	"log"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

func main() {
	fmt.Println("=== Streaming Example ===")
	fmt.Println()

	source := `
		local p = exec.spawn("sh", "-c", "for i in 1 2 3 4 5; do echo line_$i; done")

		for line in p:readlines() do
			print("Got:", line)
		end

		local result = p:wait()
		print("Exit code:", result.code)
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
	v.SetProcessProvider(vm.NewDefaultProcessProvider())
	stdlib.Open(v)

	_, err = v.Run(proto)
	if err != nil {
		log.Fatalf("Runtime error: %v", err)
	}
}
