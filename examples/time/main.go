// Example: High-resolution timing with time.now() and time.since()
//
// This is a non-standard extension. The time library provides
// millisecond-precision timing for benchmarking and elapsed-time
// measurement from Lua.
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
	source := `
		-- time.now() returns current time in milliseconds
		local start = time.now()
		print("start:", start, "ms")

		-- do some work
		local sum = 0
		for i = 1, 1000000 do
			sum = sum + i
		end

		-- time.since(t) returns milliseconds elapsed since t
		local elapsed = time.since(start)
		print("elapsed:", elapsed, "ms")
		print("sum:", sum)
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
	v.SetTimeProvider(vm.NewDefaultTimeProvider())
	stdlib.Open(v)

	_, err = v.Run(proto)
	if err != nil {
		log.Fatalf("Runtime error: %v", err)
	}

	fmt.Println("\ndone")
}
