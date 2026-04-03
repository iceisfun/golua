// Example: LuaPrintProvider for intercepting and redirecting Lua output
//
// This demonstrates how to route print() and warn() output through Go's
// logging infrastructure. Useful when embedding golua in applications that
// run multiple named scripts and need prefixed, structured output.
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

// LoggingPrintProvider prefixes all output with a script name and routes
// it through Go's log package.
type LoggingPrintProvider struct {
	ScriptName string
}

func (p *LoggingPrintProvider) Print(ctx context.Context, msg string) {
	log.Printf("[%s] %s", p.ScriptName, msg)
}

func (p *LoggingPrintProvider) Warn(ctx context.Context, msg string) {
	log.Printf("[%s] WARN: %s", p.ScriptName, msg)
}

// CollectingPrintProvider stores all output in slices for later inspection.
type CollectingPrintProvider struct {
	Prints []string
	Warns  []string
}

func (p *CollectingPrintProvider) Print(ctx context.Context, msg string) {
	p.Prints = append(p.Prints, msg)
}

func (p *CollectingPrintProvider) Warn(ctx context.Context, msg string) {
	p.Warns = append(p.Warns, msg)
}

func main() {
	fmt.Println("=== LuaPrintProvider Example ===")
	fmt.Println()

	source := `
		print("hello from lua")
		print("the answer is", 42)
		warn("something fishy")
		warn("@off")
		warn("this is silenced")
		warn("@on")
		warn("back online")
	`

	block, err := parser.Parse("example", source)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}
	proto, err := compiler.Compile("example", block)
	if err != nil {
		log.Fatalf("Compile error: %v", err)
	}

	// --- Demo 1: Logging provider ---
	fmt.Println("--- LoggingPrintProvider ---")
	v1 := vm.New()
	if err := v1.SetPrintProvider(&LoggingPrintProvider{ScriptName: "inventory.lua"}); err != nil {
		log.Fatalf("SetPrintProvider: %v", err)
	}
	stdlib.Open(v1)

	if _, err := v1.Run(proto); err != nil {
		log.Fatalf("Runtime error: %v", err)
	}
	fmt.Println()

	// --- Demo 2: Collecting provider ---
	fmt.Println("--- CollectingPrintProvider ---")
	collector := &CollectingPrintProvider{}
	v2 := vm.New()
	if err := v2.SetPrintProvider(collector); err != nil {
		log.Fatalf("SetPrintProvider: %v", err)
	}
	stdlib.Open(v2)

	if _, err := v2.Run(proto); err != nil {
		log.Fatalf("Runtime error: %v", err)
	}
	fmt.Printf("Collected %d prints:\n", len(collector.Prints))
	for i, line := range collector.Prints {
		fmt.Printf("  [%d] %s\n", i+1, line)
	}
	fmt.Printf("Collected %d warns:\n", len(collector.Warns))
	for i, line := range collector.Warns {
		fmt.Printf("  [%d] %s\n", i+1, line)
	}
	fmt.Println()

	// --- Demo 3: Per-VM warn isolation ---
	fmt.Println("--- Per-VM warn isolation ---")
	warnSource := `warn("@off") warn("should be silent")`
	warnBlock, _ := parser.Parse("vm1", warnSource)
	warnProto, _ := compiler.Compile("vm1", warnBlock)

	c1 := &CollectingPrintProvider{}
	c2 := &CollectingPrintProvider{}

	vm1 := vm.New()
	if err := vm1.SetPrintProvider(c1); err != nil {
		log.Fatalf("SetPrintProvider: %v", err)
	}
	stdlib.Open(vm1)

	vm2 := vm.New()
	if err := vm2.SetPrintProvider(c2); err != nil {
		log.Fatalf("SetPrintProvider: %v", err)
	}
	stdlib.Open(vm2)

	// vm1 turns off warnings
	vm1.Run(warnProto)

	// vm2 should still have warnings enabled
	checkSource := `warn("vm2 still warns")`
	checkBlock, _ := parser.Parse("vm2", checkSource)
	checkProto, _ := compiler.Compile("vm2", checkBlock)
	vm2.Run(checkProto)

	fmt.Printf("vm1 warns: %s\n", formatSlice(c1.Warns))
	fmt.Printf("vm2 warns: %s\n", formatSlice(c2.Warns))

	fmt.Println("\n=== Complete ===")
}

func formatSlice(s []string) string {
	if len(s) == 0 {
		return "(none)"
	}
	return strings.Join(s, ", ")
}
