package golua_test

import "testing"

// Binary/unary operands that are plain in-register locals are read in place
// (reference Lua's exp2anyreg) instead of being snapshotted into a temp with
// a MOVE. Beyond dropping the MOVE, this is observable: the operand register
// is read live at execution time, after the other operand's evaluation may
// have mutated the local through a shared upvalue. Reference Lua 5.5 prints
// 103 here; the old MOVE-snapshot codegen printed 4.
func TestBinopLeftLocalReadLive(t *testing.T) {
	got := runLuaCapture(t, `
local a = 1
local function f() a = 100 return 3 end
a = a + f()
print(a)`)
	if got != "103" {
		t.Fatalf("left operand snapshotted before right side ran: got %q want %q", got, "103")
	}
}

// Same live-read property with the roles reversed: the left operand is a call
// that mutates the right operand local. The right local must be read after
// the call (reference prints 57: g() yields 7, b is 50 by then).
func TestBinopRightLocalReadLive(t *testing.T) {
	got := runLuaCapture(t, `
local b = 1
local function g() b = 50 return 7 end
print(g() + b)`)
	if got != "57" {
		t.Fatalf("right operand read stale value: got %q want %q", got, "57")
	}
}

// freeReg epilogue regression guard: with in-place local operands the operand
// registers sit below live locals, so compileBinop must restore its entry
// register top rather than "freeing" down to an operand register. If it hands
// live local slots back to the allocator, the locals declared after the
// arithmetic get clobbered by later temps.
func TestBinopInPlaceOperandsKeepLocalsAlive(t *testing.T) {
	got := runLuaCapture(t, `
local function h(x) return x * 2 end
local a, b = 3, 4
local c = a * b       -- both operands in place, dest is a fresh local
local d = h(a) + b    -- call left into temp, right in place
local e = a + h(b)    -- left in place, call right into temp
b = a - b             -- dest is a live local, both operands in place
a = h(a) - a          -- dest live local, call temp + in-place right
local s = 0
for i = 1, 3 do
  s = s + a * i + (c - d) * e  -- nested binops under a loop variable
end
print(a, b, c, d, e, s)`)
	want := "3\t-1\t12\t10\t11\t84"
	if got != want {
		t.Fatalf("locals clobbered by binop register epilogue: got %q want %q", got, want)
	}
}

// Unary operators also read plain local operands in place (LEN t reads t's
// register directly instead of MOVE+LEN). Guard values and aliasing when the
// destination is the operand's own register.
func TestUnopLocalReadInPlace(t *testing.T) {
	got := runLuaCapture(t, `
local t = {1, 2, 3, 4}
local n = 5
local ln = #t
n = -n
local nt = not n
local bn = ~ln
print(ln, n, nt, bn)`)
	want := "4\t-5\tfalse\t-5"
	if got != want {
		t.Fatalf("unop in-place operand: got %q want %q", got, want)
	}
}

// A short-circuit operator whose constant left operand decides it (`K and a`
// with K always true, `K or a` with K always false) is just `a`: reference
// goiftrue/goiffalse emit no test, so a local `a` stays in its register and is
// read live like `(a)`. golua used to materialise the constant, test it and
// copy `a` into a temp, snapshotting it before the other operand ran. The
// last two lines are controls whose left operand does not fold away, where
// reference snapshots too. Expected output is reference Lua 5.5's.
func TestConstShortCircuitOperandReadLive(t *testing.T) {
	got := runLuaCapture(t, `
local K <const> = true
local N <const> = nil
local a = 1
local function seta() a = 10 return 0 end
local out = {}
a = 1 out[#out+1] = (nil or a) + seta()
a = 1 out[#out+1] = (true and a) + seta()
a = 1 out[#out+1] = (false or a) * (seta() + 1)
a = 1 out[#out+1] = (K and a) + seta()
a = 1 out[#out+1] = (N or a) + seta()
a = 1 out[#out+1] = (not nil and a) + seta()
a = 1 out[#out+1] = ('s' and a) // (seta() + 1)
a = 1 out[#out+1] = (nil or nil or a) + seta()
a = 1 out[#out+1] = (true and (nil or a)) + seta()
a = 1 out[#out+1] = (nil and a or a) + seta()
a = 1 out[#out+1] = (a or nil) + seta()
print(table.concat(out, " "))`)
	want := "10 10 10 10 10 10 10 10 10 1 1"
	if got != want {
		t.Fatalf("constant and/or operand snapshotted: got %q want %q", got, want)
	}
}

// The same reduction applies to the table and key of an indexed assignment
// target: the store reads the live local after the right-hand side ran.
func TestConstShortCircuitAssignTargetReadLive(t *testing.T) {
	got := runLuaCapture(t, `
local u = {}
local old = u
local function r() u = {} return 9 end
(nil or u).k = r()
local k = "a"
local v = {}
local function s() k = "b" return 5 end
v[(true and k)] = s()
print(old.k, u.k, v.a, v.b)`)
	want := "nil\t9\tnil\t5"
	if got != want {
		t.Fatalf("constant and/or assignment operand: got %q want %q", got, want)
	}
}

// Which and/or operands count as deciding constants follows reference
// constfolding: it folds infinite powers, bitwise operators on floats with an
// exact integer value, parenthesised constants and inlined <const> locals, but
// never a float zero or NaN result, so -0.0 is a UNM at run time and neither
// decides the operator nor makes an inlined <const>. Each line reads `a` live
// (10) exactly when reference folds the left operand and snapshots it (1)
// otherwise. Expected output is reference Lua 5.5's.
func TestConstShortCircuitFoldsLikeReference(t *testing.T) {
	got := runLuaCapture(t, `
local K <const> = 1
local Z <const> = -0.0
local W <const> = 0.0
local P <const> = -(2)
local KK <const> = K + 1
local a = 1
local function seta() a = 10 return 0 end
local out = {}
a = 1 out[#out+1] = (-0.0 and a) + seta()
a = 1 out[#out+1] = (Z and a) + seta()
a = 1 out[#out+1] = (not -0.0 or a) + seta()
a = 1 out[#out+1] = (-W and a) + seta()
a = 1 out[#out+1] = ((0/0) and a) + seta()
a = 1 out[#out+1] = (1.0 - 1.0 and a) + seta()
a = 1 out[#out+1] = (W and a) + seta()
a = 1 out[#out+1] = (2^10000 and a) + seta()
a = 1 out[#out+1] = (1.0|2 and a) + seta()
a = 1 out[#out+1] = (~1.0 and a) + seta()
a = 1 out[#out+1] = ((2)+(3) and a) + seta()
a = 1 out[#out+1] = (P and a) + seta()
a = 1 out[#out+1] = (K+1 and a) + seta()
a = 1 out[#out+1] = (KK and a) + seta()
print(table.concat(out, " "), 1/Z, 1.0|2, math.type(2.0<<1), -(2^10000))`)
	want := "1 1 1 1 1 1 10 10 10 10 10 10 10 10\t-inf\t3\tinteger\t-inf"
	if got != want {
		t.Fatalf("constant folding of and/or operands: got %q want %q", got, want)
	}
}
