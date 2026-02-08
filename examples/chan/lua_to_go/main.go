// Example: Lua sends messages to Go via a channel.
//
// Lua produces messages into a buffered channel, and Go reads them
// after the script completes.
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
	fmt.Println("=== Lua-to-Go Channel Example ===")
	fmt.Println()

	provider := vm.NewDefaultChanProvider()
	results := provider.NewChannel(10) // buffered so Lua won't block

	source := `
		print("Lua: sending results...")
		for i = 1, 5 do
			local msg = "result-" .. tostring(i)
			results:send(msg)
			print("Lua: sent", msg)
		end
		results:close()
		print("Lua: done sending.")
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
	v.SetChanProvider(provider)
	stdlib.Open(v)
	v.SetGlobal("results", stdlib.WrapChannel(v, results))

	_, err = v.Run(proto)
	if err != nil {
		log.Fatalf("Runtime error: %v", err)
	}

	// Go reads the messages after script completion
	fmt.Println("\nGo: reading results from channel...")
	for {
		val, ok, _ := results.Recv(nil)
		if !ok {
			break
		}
		fmt.Printf("Go: received: %s\n", val.AsString())
	}

	fmt.Println("\n=== Complete ===")
}
