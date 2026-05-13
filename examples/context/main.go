// Example: Context cancellation stops a runaway Lua script.
//
// A game server runs Lua AI scripts for NPCs. When the game round ends,
// the server cancels the context and all running scripts stop promptly,
// even if they contain infinite loops.
//
// Run: go run ./examples/context
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/iceisfun/golua/v1/compiler"
	"github.com/iceisfun/golua/v1/parser"
	"github.com/iceisfun/golua/v1/stdlib"
	"github.com/iceisfun/golua/v1/vm"
)

func main() {
	fmt.Println("=== Context Cancellation Example ===")
	fmt.Println()

	// A degenerate NPC AI script that never terminates.
	// Without cancellation this would run forever.
	npcScript := `
		local tick = 0
		local last = os.clock()

		while true do
			tick = tick + 1

			if tick % 100000 == 0 then
				local now = os.clock()
				print(string.format("tick=%d dt=%.3f", tick, now - last))
				last = now
			end
		end
	`

	block, err := parser.Parse("npc_ai", npcScript)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}
	proto, err := compiler.Compile("npc_ai", block)
	if err != nil {
		log.Fatalf("Compile error: %v", err)
	}

	// The game round lasts 500ms. When it ends, all NPC scripts must stop.
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	v := vm.New(vm.WithContext(ctx))
	if err := v.SetOsProvider(vm.NewDefaultOsProvider()); err != nil {
		log.Fatalf("SetOsProvider: %v", err)
	}
	stdlib.Open(v)

	fmt.Println("Game round started — NPC AI script running...")
	start := time.Now()

	_, err = v.Run(proto)

	elapsed := time.Since(start)
	fmt.Printf("\nScript stopped after %v\n", elapsed.Round(time.Millisecond))

	if err != nil {
		fmt.Printf("Exit reason: %v\n", err)
	}

	// Clean up providers
	v.Close(context.Background())

	fmt.Println("\n=== Game round over ===")
}
