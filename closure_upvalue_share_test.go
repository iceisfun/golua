package golua_test

import "testing"

// A closure whose only upvalue is a fresh stack capture is allocated fused with
// that upvalue. The fusion must stay invisible to Lua semantics: an upvalue is
// a shared variable, not a per-closure copy, so a second closure capturing the
// same local has to bind the *same* upvalue rather than get a new one carved
// out of its own block.
func TestFusedClosureSharesUpvalueWithSibling(t *testing.T) {
	got := runLuaCapture(t, `
local function counter()
  local n = 0
  local inc = function() n = n + 1 end
  local get = function() return n end
  return inc, get
end
local inc, get = counter()
inc(); inc(); inc()
print(get())`)
	if got != "3" {
		t.Fatalf("sibling closures did not share the captured local: got %q want %q", got, "3")
	}
}

// The sharing must hold in the other creation order too: here the reader is
// built first and takes the fused slot, and the writer must reuse it.
func TestFusedClosureSharesUpvalueReverseOrder(t *testing.T) {
	got := runLuaCapture(t, `
local n = 0
local get = function() return n end
local inc = function() n = n + 1 end
inc(); inc()
print(get())`)
	if got != "2" {
		t.Fatalf("reader closure captured a private copy: got %q want %q", got, "2")
	}
}

// Fusing must not merge distinct variables: closures over the same local share
// one cell, closures over different locals get separate ones. Mutating x through
// one fused closure has to be visible to the other closure over x, and invisible
// to the closure over y.
func TestFusedClosureDistinctLocalsStayDistinct(t *testing.T) {
	got := runLuaCapture(t, `
local x, y = 1, 2
local bumpX = function() x = x + 10 end
local readX = function() return x end
local readY = function() return y end
bumpX()
print(readX(), readY())`)
	if got != "11\t2" {
		t.Fatalf("fused closures merged distinct locals: got %q want %q", got, "11\t2")
	}
}

// Each loop iteration declares a fresh local, so every closure created in the
// loop must capture its own variable. This is the shape that takes the fused
// path most often, and getting it wrong would make every closure share one cell.
func TestFusedClosurePerIterationCapture(t *testing.T) {
	got := runLuaCapture(t, `
local fs = {}
for i = 1, 5 do fs[i] = function() return i end end
local sum = 0
for i = 1, 5 do sum = sum + fs[i]() end
print(sum, fs[1](), fs[5]())`)
	if got != "15\t1\t5" {
		t.Fatalf("loop closures did not capture per-iteration locals: got %q want %q", got, "15\t1\t5")
	}
}

// A fused upvalue is still an *open* upvalue while its frame is live: writes
// through the closure must be visible to the enclosing function reading the
// plain local, and vice versa, until the scope closes.
func TestFusedClosureOpenUpvalueAliasesLocal(t *testing.T) {
	got := runLuaCapture(t, `
local function outer()
  local v = 10
  local bump = function() v = v * 2 end
  bump()
  v = v + 1
  bump()
  return v
end
print(outer())`)
	if got != "42" {
		t.Fatalf("fused upvalue stopped aliasing the live local: got %q want %q", got, "42")
	}
}

// Sharing survives closing: once the enclosing frame exits, the single shared
// upvalue is closed once and both closures keep observing the same cell.
func TestFusedClosureSharedAfterClose(t *testing.T) {
	got := runLuaCapture(t, `
local function make()
  local n = 100
  return function() n = n + 1 return n end, function() return n end
end
local bump, peek = make()
bump()
bump()
print(peek())`)
	if got != "102" {
		t.Fatalf("shared upvalue was duplicated on close: got %q want %q", got, "102")
	}
}
