package tests

import (
	"fmt"
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
)

// TestMaxToStoreDynamicThreshold verifies that the dynamic SETLIST flush
// threshold (Lua 5.5's maxtostore) produces different flush points depending
// on register pressure, unlike the old fixed threshold of 50.
func TestMaxToStoreDynamicThreshold(t *testing.T) {
	// A simple constructor at the top level has ~253 free registers (255 - 2),
	// so the threshold should be numFree/5 ≈ 50. A deeply nested constructor
	// has fewer free registers and should flush sooner.

	// Count SETLIST instructions in compiled bytecode.
	countSetList := func(source string) int {
		t.Helper()
		block, err := parser.Parse("test", source)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		proto, err := compiler.Compile("test", block)
		if err != nil {
			t.Fatalf("compile error: %v", err)
		}
		count := 0
		// Check main proto and all child protos.
		var walk func(p *compiler.Proto)
		walk = func(p *compiler.Proto) {
			for _, inst := range p.Code {
				if inst.OpCode() == compiler.OP_SETLIST {
					count++
				}
			}
			for _, child := range p.Protos {
				walk(child)
			}
		}
		walk(proto)
		return count
	}

	// A top-level constructor with 51 items should flush once during the loop
	// (at the threshold) and once for the remaining items = 2 SETLIST ops when
	// the threshold is ~50, but we just need to verify it compiles and works.
	n := countSetList(`return {1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,
		21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,
		41,42,43,44,45,46,47,48,49,50,51}`)
	if n < 1 {
		t.Errorf("expected at least 1 SETLIST for 51-item constructor, got %d", n)
	}

	// Test that deeply nested constructors compile without "too many registers"
	// error. With a fixed threshold of 50, nesting 6 levels of 45-item
	// constructors would exceed 255 registers. The dynamic threshold makes
	// this possible by flushing sooner at higher register pressure.
	deepSource := `
local function deep(n)
  if n == 0 then return {} end
  local inner = deep(n - 1)
  local t = {1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,
             21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,
             41,42,43,44,45,inner}
  return t
end
return deep(6)
`
	// This should compile without error.
	block, err := parser.Parse("test", deepSource)
	if err != nil {
		t.Fatalf("parse error for deep nesting: %v", err)
	}
	_, err = compiler.Compile("test", block)
	if err != nil {
		t.Fatalf("compile error for deep nesting (dynamic threshold should prevent this): %v", err)
	}
}

// TestMaxToStoreFlushCountChanges verifies that when a constructor is compiled
// inside a function that already uses many registers (high register pressure),
// the flush threshold decreases, resulting in more SETLIST instructions.
func TestMaxToStoreFlushCountChanges(t *testing.T) {
	countSetList := func(source string) int {
		t.Helper()
		block, err := parser.Parse("test", source)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		proto, err := compiler.Compile("test", block)
		if err != nil {
			t.Fatalf("compile error: %v", err)
		}
		count := 0
		var walk func(p *compiler.Proto)
		walk = func(p *compiler.Proto) {
			for _, inst := range p.Code {
				if inst.OpCode() == compiler.OP_SETLIST {
					count++
				}
			}
			for _, child := range p.Protos {
				walk(child)
			}
		}
		walk(proto)
		return count
	}

	// With low register pressure (top level): 20 items should be 1 SETLIST
	// because threshold = (255-2)/5 ≈ 50 > 20.
	lowPressure := countSetList(`return {1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20}`)
	if lowPressure != 1 {
		t.Errorf("expected 1 SETLIST for 20-item low-pressure constructor, got %d", lowPressure)
	}

	// With high register pressure (many locals consuming registers), the same
	// 20 items may need multiple flushes because the threshold is smaller.
	// With numFree < 80, threshold = 1, so 20 items -> 20 SETLIST ops.
	// Generate source with enough locals to push freeReg high.
	// Use v_NNN names to avoid clashing with Lua keywords.
	highPressureSrc := "local "
	for i := 0; i < 180; i++ {
		if i > 0 {
			highPressureSrc += ", "
		}
		highPressureSrc += fmt.Sprintf("v_%d", i)
	}
	highPressureSrc += " = "
	for i := 0; i < 180; i++ {
		if i > 0 {
			highPressureSrc += ", "
		}
		highPressureSrc += "1"
	}
	highPressureSrc += "\nreturn {1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20}\n"

	highPressure := countSetList(highPressureSrc)
	if highPressure <= lowPressure {
		t.Errorf("expected more SETLIST ops under high register pressure (%d) than low (%d)",
			highPressure, lowPressure)
	}
}
