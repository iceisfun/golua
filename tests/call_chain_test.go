package tests

import (
	"strings"
	"testing"
)

// TestCallChainLimit_1Level verifies a single __call level works.
func TestCallChainLimit_1Level(t *testing.T) {
	_, output, err := runLua(t, `
		local t = setmetatable({}, {__call = function(self) return "ok" end})
		print(t())
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output) != 1 || output[0] != "ok" {
		t.Fatalf("expected [ok], got %v", output)
	}
}

// TestCallChainLimit_15Levels verifies that 15 levels (the maximum) work.
func TestCallChainLimit_15Levels(t *testing.T) {
	_, output, err := runLua(t, `
		local chain = {}
		for i = 1, 15 do
			local prev = chain[#chain]
			local t = setmetatable({}, {__call = prev or function() return "deep" end})
			chain[#chain+1] = t
		end
		print(chain[15]())
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output) != 1 || output[0] != "deep" {
		t.Fatalf("expected [deep], got %v", output)
	}
}

// TestCallChainLimit_16Levels verifies that 16 levels triggers an error.
func TestCallChainLimit_16Levels(t *testing.T) {
	_, output, err := runLua(t, `
		local chain = {}
		for i = 1, 16 do
			local prev = chain[#chain]
			local t = setmetatable({}, {__call = prev or function() return "deep" end})
			chain[#chain+1] = t
		end
		local ok, err = pcall(chain[16])
		print(ok)
		print(err)
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output) < 2 {
		t.Fatalf("expected 2 output lines, got %v", output)
	}
	if output[0] != "false" {
		t.Fatalf("expected pcall to fail, got %q", output[0])
	}
	if !strings.Contains(output[1], "'__call' chain too long") {
		t.Fatalf("expected '__call' chain too long error, got %q", output[1])
	}
}

// TestCallChainLimit_SelfReferencing verifies a self-referencing __call errors.
func TestCallChainLimit_SelfReferencing(t *testing.T) {
	_, output, err := runLua(t, `
		local t = {}
		setmetatable(t, {__call = t})
		local ok, err = pcall(t)
		print(ok)
		print(err)
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output) < 2 {
		t.Fatalf("expected 2 output lines, got %v", output)
	}
	if output[0] != "false" {
		t.Fatalf("expected pcall to fail, got %q", output[0])
	}
	if !strings.Contains(output[1], "'__call' chain too long") {
		t.Fatalf("expected '____call' chain too long error, got %q", output[1])
	}
}

// TestCallChainLimit_TailCall verifies the limit applies in tail-call position.
func TestCallChainLimit_TailCall(t *testing.T) {
	_, output, err := runLua(t, `
		local chain = {}
		for i = 1, 16 do
			local prev = chain[#chain]
			local t = setmetatable({}, {__call = prev or function() return "deep" end})
			chain[#chain+1] = t
		end
		local ok, err = pcall(function() return chain[16]() end)
		print(ok)
		print(err)
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output) < 2 {
		t.Fatalf("expected 2 output lines, got %v", output)
	}
	if output[0] != "false" {
		t.Fatalf("expected pcall to fail, got %q", output[0])
	}
	if !strings.Contains(output[1], "'__call' chain too long") {
		t.Fatalf("expected '__call' chain too long error, got %q", output[1])
	}
}

// TestCallChainLimit_14Levels verifies that 14 levels work fine.
func TestCallChainLimit_14Levels(t *testing.T) {
	_, output, err := runLua(t, `
		local chain = {}
		for i = 1, 14 do
			local prev = chain[#chain]
			local t = setmetatable({}, {__call = prev or function() return "ok14" end})
			chain[#chain+1] = t
		end
		print(chain[14]())
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output) != 1 || output[0] != "ok14" {
		t.Fatalf("expected [ok14], got %v", output)
	}
}

// TestCallChainLimit_IndependentCalls verifies that the chain limit is per-call,
// not cumulative across multiple calls.
func TestCallChainLimit_IndependentCalls(t *testing.T) {
	_, output, err := runLua(t, `
		local chain = {}
		for i = 1, 15 do
			local prev = chain[#chain]
			local t = setmetatable({}, {__call = prev or function() return "ok" end})
			chain[#chain+1] = t
		end
		-- Two independent calls at depth 15 should each succeed
		print(chain[15]())
		print(chain[15]())
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output) != 2 || output[0] != "ok" || output[1] != "ok" {
		t.Fatalf("expected [ok ok], got %v", output)
	}
}
