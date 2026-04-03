// Example: Go pushes events to Lua via a channel.
//
// A Go goroutine sends timestamped events, and Lua consumes them
// until the channel is closed.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

func main() {
	fmt.Println("=== Go-to-Lua Channel Example ===")
	fmt.Println()

	provider := vm.NewDefaultChanProvider()
	events := provider.NewChannel(context.Background(), 0) // unbuffered for synchronous handoff

	// Go goroutine: send 5 events then close
	go func() {
		for i := 1; i <= 5; i++ {
			msg := fmt.Sprintf("event-%d at %s", i, time.Now().Format("15:04:05.000"))
			events.Send(nil, vm.NewString(msg))
		}
		events.Close()
	}()

	source := `
		print("Lua: waiting for events...")
		while true do
			local msg, ok = events:recv()
			if not ok then
				print("Lua: channel closed, done.")
				break
			end
			print("Lua: received:", msg)
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
	if err := v.SetChanProvider(provider); err != nil {
		log.Fatalf("SetChanProvider: %v", err)
	}
	stdlib.Open(v)
	v.SetGlobal("events", stdlib.WrapChannel(v, events))

	_, err = v.Run(proto)
	if err != nil {
		log.Fatalf("Runtime error: %v", err)
	}

	fmt.Println("\n=== Complete ===")
}
