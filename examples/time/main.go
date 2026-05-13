// Example: High-resolution timing with time.now(), time.since(), time.tick(), and time.once()
//
// This is a non-standard extension. The time library provides
// millisecond-precision timing for benchmarking, elapsed-time
// measurement, and periodic triggers from Lua.
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
		print()

		-- time.tick(ms) returns true once per interval, false otherwise
		-- useful for periodic logic inside hot loops
		local ticks = 0
		for i = 1, 5000000 do
			if time.tick(50) then
				ticks = ticks + 1
				print("tick #" .. ticks .. " at i=" .. i)
			end
		end
		print("total ticks:", ticks)
		print()

		-- time.once() returns true on first call, false thereafter
		-- useful for one-time initialization inside loops
		local inits = 0
		for i = 1, 5 do
			if time.once() then
				inits = inits + 1
				print("initialized at i=" .. i)
			end
		end
		print("init count:", inits) -- always 1
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
	if err := v.SetTimeProvider(vm.NewDefaultTimeProvider()); err != nil {
		log.Fatalf("SetTimeProvider: %v", err)
	}
	stdlib.Open(v)

	_, err = v.Run(proto)
	if err != nil {
		log.Fatalf("Runtime error: %v", err)
	}

	fmt.Println("\ndone")
}
