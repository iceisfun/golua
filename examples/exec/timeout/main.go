// Example: Process timeout and kill.
//
// Demonstrates timed wait and process termination.
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
	fmt.Println("=== Timeout Example ===")
	fmt.Println()

	source := `
		local p = exec.spawn("sleep", "10")

		local result, done = p:wait(500)  -- 500ms timeout

		if not done then
			print("Process did not finish in time, killing...")
			p:kill()
			local r = p:wait()
			print("Killed, success:", r.success)
		else
			print("Process finished, code:", result.code)
		end
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
