// Example: Multiple Go producers, Lua consumer using chan.select.
//
// Three Go goroutines send messages on separate channels.
// Lua uses chan.select to receive from whichever channel has data.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

func main() {
	fmt.Println("=== Multi-Producer chan.select Example ===")
	fmt.Println()

	provider := vm.NewDefaultChanProvider()
	ch1 := provider.NewChannel(context.Background(), 0) // unbuffered for synchronous handoff
	ch2 := provider.NewChannel(context.Background(), 0)
	ch3 := provider.NewChannel(context.Background(), 0)
	done := provider.NewChannel(context.Background(), 0)

	source := `
		local names = {"alpha", "beta", "gamma"}
		local total = 0

		-- Receive until 'done' signal
		while true do
			-- Select across data channels + done channel (4th)
			local idx, val, ok = chan.select(ch1, ch2, ch3, done)
			if idx == 4 then
				print("Lua: received stop signal")
				break
			end
			if ok then
				total = total + 1
				print("Lua: [" .. names[idx] .. "] " .. val)
			end
		end

		print("\nLua: received " .. total .. " messages total")
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
	v.SetGlobal("ch1", stdlib.WrapChannel(v, ch1))
	v.SetGlobal("ch2", stdlib.WrapChannel(v, ch2))
	v.SetGlobal("ch3", stdlib.WrapChannel(v, ch3))
	v.SetGlobal("done", stdlib.WrapChannel(v, done))

	// Producers send messages sequentially, then signal done.
	// Unbuffered channels mean each Send blocks until Lua reads.
	go func() {
		produce := func(ch *vm.LuaChannel, name string, count int) {
			for i := 1; i <= count; i++ {
				ch.Send(nil, vm.NewString(fmt.Sprintf("%s-%d", name, i)))
			}
		}

		produce(ch1, "alpha", 3)
		produce(ch2, "beta", 2)
		produce(ch3, "gamma", 1)

		// Signal Lua to stop
		done.Send(nil, vm.NewInt(1))
	}()

	_, err = v.Run(proto)
	if err != nil {
		log.Fatalf("Runtime error: %v", err)
	}

	fmt.Println("\n=== Complete ===")
}
