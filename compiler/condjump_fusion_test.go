// Package compiler_test holds the condition/jump fusion tests. It is an
// external test package because it drives the VM (which imports compiler) to
// verify that the immediate-compare opcodes the compiler now emits are
// executed correctly.
package compiler_test

import (
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func compileSrc(t *testing.T, src string) *compiler.Proto {
	t.Helper()
	block, err := parser.Parse("@test", src)
	if err != nil {
		t.Fatalf("parse error: %v\nsource:\n%s", err, src)
	}
	proto, err := compiler.Compile("@test", block)
	if err != nil {
		t.Fatalf("compile error: %v\nsource:\n%s", err, src)
	}
	return proto
}

// opNames renders a proto's instruction stream as opcode mnemonics, which is
// what the fusion shape assertions compare against.
func opNames(p *compiler.Proto) []string {
	names := make([]string, 0, len(p.Code))
	for _, inst := range p.Code {
		names = append(names, inst.OpCode().String())
	}
	return names
}

// runSrc compiles and runs src on a fresh VM with the standard library open.
func runSrc(t *testing.T, src string) ([]vm.Value, error) {
	t.Helper()
	p := compileSrc(t, src)
	v := vm.New()
	v.SetDebugProvider(vm.NewDefaultDebugProvider())
	stdlib.Open(v)
	return v.Run(p)
}

// runSrcOK runs src and fails the test on error.
func runSrcOK(t *testing.T, src string) []vm.Value {
	t.Helper()
	res, err := runSrc(t, src)
	if err != nil {
		t.Fatalf("run error: %v\nsource:\n%s", err, src)
	}
	return res
}

func firstString(t *testing.T, res []vm.Value) string {
	t.Helper()
	if len(res) == 0 {
		t.Fatalf("expected a result, got none")
	}
	return res[0].String()
}

// ---------------------------------------------------------------------------
// Part 1: direct VM verification of the immediate-compare opcodes.
//
// These opcodes (EQI/EQK/LTI/LEI/GTI/GEI) were dispatched by the VM but never
// emitted by the compiler, so their implementations had never been exercised.
// A sibling pair (OP_SHLI/OP_SHRI) is implemented swapped relative to the
// opcode table for exactly that reason, so every immediate compare is proven
// here from a HAND-BUILT prototype — no compiler involvement — before the
// compiler is allowed to emit it.
// ---------------------------------------------------------------------------

// immCompareProto builds a prototype that loads K[0] into R0, runs
// `op R0, b, k`, and returns true when the comparison instruction skipped the
// following JMP (i.e. the tested condition held with the given k sense).
//
//	0: LOADK     R0, K[0]
//	1: op        R0, b, k     ; pc++ (skip 2) when result != k
//	2: JMP       -> 5
//	3: LOADTRUE  R1
//	4: JMP       -> 6
//	5: LOADFALSE R1
//	6: RETURN1   R1
func immCompareProto(op compiler.OpCode, b, k int, consts []compiler.Value) *compiler.Proto {
	code := []compiler.Instruction{
		compiler.ABx(compiler.OP_LOADK, 0, 0),
		compiler.ABC(op, 0, b, 0, k),
		compiler.SJ(compiler.OP_JMP, 2, 0),
		compiler.ABC(compiler.OP_LOADTRUE, 1, 0, 0, 0),
		compiler.SJ(compiler.OP_JMP, 1, 0),
		compiler.ABC(compiler.OP_LOADFALSE, 1, 0, 0, 0),
		compiler.ABC(compiler.OP_RETURN1, 1, 0, 0, 0),
	}
	lines := make([]int, len(code))
	for i := range lines {
		lines[i] = 1
	}
	return &compiler.Proto{
		Source:    "=cmp",
		LineDef:   1,
		LastLine:  1,
		MaxStack:  2,
		Code:      code,
		Lines:     lines,
		Constants: consts,
	}
}

// runImmCompare executes a hand-built immediate compare and reports whether
// the condition held (k=0 sense: "true when the comparison is true").
func runImmCompare(t *testing.T, op compiler.OpCode, imm int, consts []compiler.Value) bool {
	t.Helper()
	b := imm + compiler.OffsetSC
	if op == compiler.OP_EQK {
		b = imm // for EQK the B field is a constant index, not a biased sC
	}
	v := vm.New()
	res, err := v.Run(immCompareProto(op, b, 0, consts))
	if err != nil {
		t.Fatalf("%s: run error: %v", op.String(), err)
	}
	if len(res) != 1 {
		t.Fatalf("%s: expected 1 result, got %d", op.String(), len(res))
	}
	return res[0].ToBool()
}

func TestVMImmediateCompareOpcodes(t *testing.T) {
	// Reference semantics (lua-5.5.0/src/lvm.c, op_orderI / OP_EQI):
	//   LTI: R[A] <  sB      LEI: R[A] <= sB
	//   GTI: R[A] >  sB      GEI: R[A] >= sB
	//   EQI: R[A] == sB (numbers only; other types are never equal)
	// Every expectation below was cross-checked against /usr/bin/lua5.5.0
	// (see TestImmediateCompareDifferential for the source-level version).
	iv := compiler.IntValue
	fv := compiler.FloatValue

	tests := []struct {
		name string
		op   compiler.OpCode
		val  compiler.Value
		imm  int
		want bool
	}{
		// --- LTI ---
		{"LTI int less", compiler.OP_LTI, iv(3), 5, true},
		{"LTI int equal", compiler.OP_LTI, iv(5), 5, false},
		{"LTI int greater", compiler.OP_LTI, iv(7), 5, false},
		{"LTI negative imm", compiler.OP_LTI, iv(-9), -5, true},
		{"LTI negative imm eq", compiler.OP_LTI, iv(-5), -5, false},
		{"LTI negative val pos imm", compiler.OP_LTI, iv(-1), 0, true},
		{"LTI sC low bound", compiler.OP_LTI, iv(-128), -127, true},
		{"LTI sC low bound eq", compiler.OP_LTI, iv(-127), -127, false},
		{"LTI sC high bound", compiler.OP_LTI, iv(126), 127, true},
		{"LTI sC high bound eq", compiler.OP_LTI, iv(127), 127, false},
		{"LTI maxint", compiler.OP_LTI, iv(math.MaxInt64), 127, false},
		{"LTI minint", compiler.OP_LTI, iv(math.MinInt64), -127, true},
		{"LTI float less", compiler.OP_LTI, fv(4.5), 5, true},
		{"LTI float equal", compiler.OP_LTI, fv(5.0), 5, false},
		{"LTI float greater", compiler.OP_LTI, fv(5.5), 5, false},
		{"LTI +inf", compiler.OP_LTI, fv(math.Inf(1)), 127, false},
		{"LTI -inf", compiler.OP_LTI, fv(math.Inf(-1)), -127, true},
		{"LTI NaN", compiler.OP_LTI, fv(math.NaN()), 0, false},

		// --- LEI ---
		{"LEI int less", compiler.OP_LEI, iv(3), 5, true},
		{"LEI int equal", compiler.OP_LEI, iv(5), 5, true},
		{"LEI int greater", compiler.OP_LEI, iv(7), 5, false},
		{"LEI negative", compiler.OP_LEI, iv(-5), -5, true},
		{"LEI negative less", compiler.OP_LEI, iv(-6), -5, true},
		{"LEI sC low bound", compiler.OP_LEI, iv(-127), -127, true},
		{"LEI sC high bound", compiler.OP_LEI, iv(127), 127, true},
		{"LEI float equal", compiler.OP_LEI, fv(5.0), 5, true},
		{"LEI float just above", compiler.OP_LEI, fv(5.000001), 5, false},
		{"LEI NaN", compiler.OP_LEI, fv(math.NaN()), 0, false},

		// --- GTI ---
		{"GTI int greater", compiler.OP_GTI, iv(7), 5, true},
		{"GTI int equal", compiler.OP_GTI, iv(5), 5, false},
		{"GTI int less", compiler.OP_GTI, iv(3), 5, false},
		{"GTI negative", compiler.OP_GTI, iv(-3), -5, true},
		{"GTI negative eq", compiler.OP_GTI, iv(-5), -5, false},
		{"GTI sC low bound", compiler.OP_GTI, iv(-126), -127, true},
		{"GTI sC high bound", compiler.OP_GTI, iv(127), 127, false},
		{"GTI maxint", compiler.OP_GTI, iv(math.MaxInt64), 127, true},
		{"GTI float greater", compiler.OP_GTI, fv(5.5), 5, true},
		{"GTI float equal", compiler.OP_GTI, fv(5.0), 5, false},
		{"GTI NaN", compiler.OP_GTI, fv(math.NaN()), 0, false},

		// --- GEI ---
		{"GEI int greater", compiler.OP_GEI, iv(7), 5, true},
		{"GEI int equal", compiler.OP_GEI, iv(5), 5, true},
		{"GEI int less", compiler.OP_GEI, iv(3), 5, false},
		{"GEI negative eq", compiler.OP_GEI, iv(-5), -5, true},
		{"GEI negative less", compiler.OP_GEI, iv(-6), -5, false},
		{"GEI sC low bound", compiler.OP_GEI, iv(-127), -127, true},
		{"GEI sC high bound", compiler.OP_GEI, iv(127), 127, true},
		{"GEI minint", compiler.OP_GEI, iv(math.MinInt64), -127, false},
		{"GEI float equal", compiler.OP_GEI, fv(5.0), 5, true},
		{"GEI float below", compiler.OP_GEI, fv(4.999), 5, false},
		{"GEI NaN", compiler.OP_GEI, fv(math.NaN()), 0, false},

		// --- EQI ---
		{"EQI int equal", compiler.OP_EQI, iv(5), 5, true},
		{"EQI int differ", compiler.OP_EQI, iv(6), 5, false},
		{"EQI negative equal", compiler.OP_EQI, iv(-5), -5, true},
		{"EQI negative differ", compiler.OP_EQI, iv(-5), 5, false},
		{"EQI zero", compiler.OP_EQI, iv(0), 0, true},
		{"EQI sC low bound", compiler.OP_EQI, iv(-127), -127, true},
		{"EQI sC high bound", compiler.OP_EQI, iv(127), 127, true},
		{"EQI maxint", compiler.OP_EQI, iv(math.MaxInt64), 127, false},
		// Lua: 5.0 == 5 is true (mixed int/float equality compares values).
		{"EQI float equal int", compiler.OP_EQI, fv(5.0), 5, true},
		{"EQI float negative", compiler.OP_EQI, fv(-5.0), -5, true},
		{"EQI float fraction", compiler.OP_EQI, fv(5.5), 5, false},
		{"EQI NaN", compiler.OP_EQI, fv(math.NaN()), 0, false},
		// Non-numbers are never equal to a number, and never invoke __eq.
		{"EQI string", compiler.OP_EQI, compiler.StringValue("5"), 5, false},
		{"EQI nil", compiler.OP_EQI, compiler.NilValue(), 0, false},
		{"EQI true", compiler.OP_EQI, compiler.BoolValue(true), 1, false},
		{"EQI false", compiler.OP_EQI, compiler.BoolValue(false), 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runImmCompare(t, tt.op, tt.imm, []compiler.Value{tt.val})
			if got != tt.want {
				t.Errorf("%s with value %v, imm %d: got %v, want %v",
					tt.op.String(), tt.val, tt.imm, got, tt.want)
			}
		})
	}
}

func TestVMEQKOpcode(t *testing.T) {
	// EQK compares R[A] against K[B]. B is a constant index, so the compared
	// value lives at K[0] and the compare constant at K[1].
	tests := []struct {
		name  string
		val   compiler.Value
		konst compiler.Value
		want  bool
	}{
		{"equal strings", compiler.StringValue("foo"), compiler.StringValue("foo"), true},
		{"different strings", compiler.StringValue("foo"), compiler.StringValue("bar"), false},
		{"prefix", compiler.StringValue("fo"), compiler.StringValue("foo"), false},
		{"empty vs empty", compiler.StringValue(""), compiler.StringValue(""), true},
		{"embedded NUL equal", compiler.StringValue("a\x00b"), compiler.StringValue("a\x00b"), true},
		{"embedded NUL differ", compiler.StringValue("a\x00b"), compiler.StringValue("a\x00c"), false},
		{"number vs string", compiler.IntValue(5), compiler.StringValue("5"), false},
		{"nil vs string", compiler.NilValue(), compiler.StringValue("x"), false},
		{"bool vs string", compiler.BoolValue(true), compiler.StringValue("true"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runImmCompare(t, compiler.OP_EQK, 1,
				[]compiler.Value{tt.val, tt.konst})
			if got != tt.want {
				t.Errorf("EQK %v == %v: got %v, want %v", tt.val, tt.konst, got, tt.want)
			}
		})
	}
}

// TestVMImmediateCompareKSense pins the k-flag convention shared with OP_TEST:
// the instruction skips the following instruction when result != k. The fused
// codegen relies on this being a drop-in replacement for a materialized
// boolean + OP_TEST.
func TestVMImmediateCompareKSense(t *testing.T) {
	for _, k := range []int{0, 1} {
		for _, tc := range []struct {
			val  int64
			imm  int
			cond bool
		}{{3, 5, true}, {7, 5, false}} {
			v := vm.New()
			res, err := v.Run(immCompareProto(compiler.OP_LTI, 5+compiler.OffsetSC, k,
				[]compiler.Value{compiler.IntValue(tc.val)}))
			if err != nil {
				t.Fatalf("run error: %v", err)
			}
			// The proto returns true when the JMP was skipped, i.e. when
			// cond != k.
			want := tc.cond != (k == 1)
			if got := res[0].ToBool(); got != want {
				t.Errorf("LTI %d < %d with k=%d: skipped=%v, want %v",
					tc.val, tc.imm, k, got, want)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Part 2: emitted instruction shapes.
// ---------------------------------------------------------------------------

// findChild returns the single nested prototype, so shape assertions can look
// at a function body rather than the main chunk.
func findChild(t *testing.T, p *compiler.Proto) *compiler.Proto {
	t.Helper()
	if len(p.Protos) != 1 {
		t.Fatalf("expected exactly 1 nested proto, got %d", len(p.Protos))
	}
	return p.Protos[0]
}

func TestFusedConditionShapes(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			// Before fusion: LT, JMP, LOADTRUE, JMP, LOADFALSE, TEST, JMP
			// (7 instructions) for the condition alone.
			name: "if register compare",
			src:  "local function f(a,b,c) if a < b then c = 1 end return c end",
			want: []string{"LT", "JMP", "LOADI", "MOVE", "RETURN1"},
		},
		{
			name: "if immediate equality",
			src:  "local function f(d) if d == 0 then return 1 end return 2 end",
			want: []string{"EQI", "JMP", "LOADI", "RETURN1", "LOADI", "RETURN1"},
		},
		{
			name: "if string equality",
			src:  "local function f(s) if s == 'foo' then return 1 end return 2 end",
			want: []string{"EQK", "JMP", "LOADI", "RETURN1", "LOADI", "RETURN1"},
		},
		{
			name: "while immediate compare",
			src:  "local function f(x) while x < 100 do x = x + 1 end return x end",
			want: []string{"LTI", "JMP", "ADDI", "MMBINI", "JMP", "MOVE", "RETURN1"},
		},
		{
			name: "repeat immediate compare",
			src:  "local function f(x) repeat x = x - 1 until x < 0 return x end",
			want: []string{"ADDI", "MMBINI", "LTI", "JMP", "MOVE", "RETURN1"},
		},
		{
			name: "not local flips the test sense",
			src:  "local function f(a) if not a then return 1 end return 2 end",
			want: []string{"TEST", "JMP", "LOADI", "RETURN1", "LOADI", "RETURN1"},
		},
		{
			name: "parenthesised comparison stays fused",
			src:  "local function f(a,b) if (a < b) then return 1 end return 2 end",
			want: []string{"LT", "JMP", "LOADI", "RETURN1", "LOADI", "RETURN1"},
		},
		{
			name: "constant on the left flips the mnemonic",
			src:  "local function f(x) if 5 < x then return 1 end return 2 end",
			want: []string{"GTI", "JMP", "LOADI", "RETURN1", "LOADI", "RETURN1"},
		},
		{
			name: "greater-than with constant on the right",
			src:  "local function f(x) if x > 5 then return 1 end return 2 end",
			want: []string{"GTI", "JMP", "LOADI", "RETURN1", "LOADI", "RETURN1"},
		},
		{
			name: "greater-equal with constant on the right",
			src:  "local function f(x) if x >= 5 then return 1 end return 2 end",
			want: []string{"GEI", "JMP", "LOADI", "RETURN1", "LOADI", "RETURN1"},
		},
		{
			name: "less-equal with constant on the right",
			src:  "local function f(x) if x <= 5 then return 1 end return 2 end",
			want: []string{"LEI", "JMP", "LOADI", "RETURN1", "LOADI", "RETURN1"},
		},
		{
			name: "constant on the left of <=",
			src:  "local function f(x) if 5 <= x then return 1 end return 2 end",
			want: []string{"GEI", "JMP", "LOADI", "RETURN1", "LOADI", "RETURN1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := findChild(t, compileSrc(t, tt.src))
			got := opNames(p)
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("instruction shape mismatch\n got: %v\nwant: %v\n%s",
					got, tt.want, p.DumpString())
			}
		})
	}
}

// TestFusedConditionsEmitNoBooleanMaterialization is the property the whole
// optimization rests on: a comparison used as a control-flow condition must
// never materialize a boolean.
func TestFusedConditionsEmitNoBooleanMaterialization(t *testing.T) {
	srcs := []string{
		"local function f(a,b) if a < b then return 1 end end",
		"local function f(a,b) if a <= b then return 1 end end",
		"local function f(a,b) if a == b then return 1 end end",
		"local function f(a,b) if a ~= b then return 1 end end",
		"local function f(a,b) if a > b then return 1 end end",
		"local function f(a,b) if a >= b then return 1 end end",
		"local function f(a,b) while a < b do a = a + 1 end end",
		"local function f(a,b) repeat a = a + 1 until a > b end",
		"local function f(a,b) if not (a < b) then return 1 end end",
		"local function f(a,b) if a < b then return 1 elseif a > b then return 2 end end",
	}
	for _, src := range srcs {
		p := findChild(t, compileSrc(t, src))
		for _, name := range opNames(p) {
			switch name {
			case "LOADTRUE", "LOADFALSE", "LFALSESKIP", "TEST", "TESTSET", "NOT":
				t.Errorf("%s: unexpected %s in condition codegen:\n%s", src, name, p.DumpString())
			}
		}
	}
}

// TestImmediateNotEmittedOutOfRange guards the sC encoding bound: constants
// outside [-127,127] must fall back to the register form rather than being
// truncated into the sB field.
func TestImmediateNotEmittedOutOfRange(t *testing.T) {
	for _, tc := range []struct {
		src  string
		want string
	}{
		{"local function f(x) if x < 127 then return 1 end end", "LTI"},
		{"local function f(x) if x < 128 then return 1 end end", "LT"},
		// A negated literal is a unary-minus expression, not a constant, so
		// it does not reach the immediate form (pre-existing: the same is
		// true of ADDI). Reference luac folds it and emits LTI -127.
		{"local function f(x) if x < -127 then return 1 end end", "LT"},
		{"local function f(x) if x < -128 then return 1 end end", "LT"},
		{"local function f(x) if x == 127 then return 1 end end", "EQI"},
		{"local function f(x) if x == 128 then return 1 end end", "EQ"},
		// Floats never use the integer immediate form.
		{"local function f(x) if x < 5.0 then return 1 end end", "LT"},
		{"local function f(x) if x == 5.0 then return 1 end end", "EQ"},
	} {
		p := findChild(t, compileSrc(t, tc.src))
		found := false
		for _, name := range opNames(p) {
			if name == tc.want {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: expected %s, got %v", tc.src, tc.want, opNames(p))
		}
	}
}

// ---------------------------------------------------------------------------
// Part 3: end-to-end semantics through the fused paths.
// ---------------------------------------------------------------------------

// TestImmediateCompareDifferential runs the source-level equivalents of the
// hand-built opcode tests. Every expected string below was produced by
// /usr/bin/lua5.5.0 on the identical source.
func TestImmediateCompareDifferential(t *testing.T) {
	src := `
local out = {}
local function p(x) out[#out+1] = tostring(x) end
local nan = 0/0
local vals = {3, 5, 7, -5, -127, 127, -128, 128, math.maxinteger, math.mininteger,
              4.5, 5.0, 5.5, nan, math.huge, -math.huge, "5", nil, true}
for i = 1, 19 do
  local v = vals[i]
  local ok, r
  ok, r = pcall(function() if v < 5 then return "lt" end return "-" end); p(ok and r or "err")
  ok, r = pcall(function() if v <= 5 then return "le" end return "-" end); p(ok and r or "err")
  ok, r = pcall(function() if v > 5 then return "gt" end return "-" end); p(ok and r or "err")
  ok, r = pcall(function() if v >= 5 then return "ge" end return "-" end); p(ok and r or "err")
  ok, r = pcall(function() if v == 5 then return "eq" end return "-" end); p(ok and r or "err")
  ok, r = pcall(function() if v ~= 5 then return "ne" end return "-" end); p(ok and r or "err")
  ok, r = pcall(function() if v == "5" then return "eqs" end return "-" end); p(ok and r or "err")
  ok, r = pcall(function() if -5 < v then return "rlt" end return "-" end); p(ok and r or "err")
  ok, r = pcall(function() if 127 >= v then return "rge" end return "-" end); p(ok and r or "err")
end
return table.concat(out, " ")
`
	// Reference output from lua5.5.0.
	want := "lt le - - - ne - rlt rge " + // 3
		"- le - ge eq - - rlt rge " + // 5
		"- - gt ge - ne - rlt rge " + // 7
		"lt le - - - ne - - rge " + // -5
		"lt le - - - ne - - rge " + // -127
		"- - gt ge - ne - rlt rge " + // 127
		"lt le - - - ne - - rge " + // -128
		"- - gt ge - ne - rlt - " + // 128
		"- - gt ge - ne - rlt - " + // maxinteger
		"lt le - - - ne - - rge " + // mininteger
		"lt le - - - ne - rlt rge " + // 4.5
		"- le - ge eq - - rlt rge " + // 5.0
		"- - gt ge - ne - rlt rge " + // 5.5
		"- - - - - ne - - - " + // nan
		"- - gt ge - ne - rlt - " + // huge
		"lt le - - - ne - - rge " + // -huge
		"err err err err - ne eqs err err " + // "5"
		"err err err err - ne - err err " + // nil
		"err err err err - ne - err err" // true
	got := firstString(t, runSrcOK(t, src))
	if got != want {
		t.Errorf("immediate compare differential mismatch\n got: %s\nwant: %s", got, want)
	}
}

// TestFusedComparisonMetamethods checks that comparison metamethods still fire
// with the same operands in the same order once the compare is fused and, for
// the immediate forms, that the immediate is passed in the correct position
// (reference luaT_callorderiTM flips the operands for GTI/GEI exactly as the
// register form swaps them for '>'/'>=').
func TestFusedComparisonMetamethods(t *testing.T) {
	src := `
local log = {}
local mt = {}
local function nm(v) return tostring(type(v) == "table" and v.n or v) end
mt.__lt = function(a, b) log[#log+1] = "lt(" .. nm(a) .. "," .. nm(b) .. ")" return true end
mt.__le = function(a, b) log[#log+1] = "le(" .. nm(a) .. "," .. nm(b) .. ")" return true end
mt.__eq = function(a, b) log[#log+1] = "eq" return true end
local t = setmetatable({n = "t"}, mt)
if t < 5 then end
if t <= 5 then end
if t > 5 then end
if t >= 5 then end
if 5 < t then end
if 5 <= t then end
if 5 > t then end
if 5 >= t then end
if t == 5 then log[#log+1] = "BAD-eq-fired" end
if t == "s" then log[#log+1] = "BAD-eqk-fired" end
return table.concat(log, " ")
`
	// Reference output from lua5.5.0 (t == 5 and t == "s" never call __eq:
	// __eq only fires when both operands are tables or both full userdata).
	want := "lt(t,5) le(t,5) lt(5,t) le(5,t) lt(5,t) le(5,t) lt(t,5) le(t,5)"
	if got := firstString(t, runSrcOK(t, src)); got != want {
		t.Errorf("metamethod order mismatch\n got: %s\nwant: %s", got, want)
	}
}

// TestFusedComparisonErrors checks that the error message and its operand
// order survive fusion (reference builds the immediate as the second operand
// for LTI/LEI and as the first for GTI/GEI).
func TestFusedComparisonErrors(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{"local t = {} if t < 5 then end", "attempt to compare table with number"},
		{"local t = {} if t <= 5 then end", "attempt to compare table with number"},
		{"local t = {} if t > 5 then end", "attempt to compare number with table"},
		{"local t = {} if t >= 5 then end", "attempt to compare number with table"},
		{"local t = {} if 5 < t then end", "attempt to compare number with table"},
		{"local t = {} if 5 >= t then end", "attempt to compare table with number"},
		{"local t = {} local u = {} if t < u then end", "attempt to compare two table values"},
		{"local s = 'x' if s < 5 then end", "attempt to compare string with number"},
	}
	for _, tc := range cases {
		_, err := runSrc(t, tc.src)
		if err == nil {
			t.Errorf("%s: expected an error", tc.src)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s:\n got: %v\nwant substring: %s", tc.src, err, tc.want)
		}
	}
}

// TestFusedNaNSemantics locks the NaN invariant: `a < b` and `a >= b` must be
// false simultaneously, on both the register and the immediate paths.
func TestFusedNaNSemantics(t *testing.T) {
	src := `
local nan = 0/0
local out = {}
local function p(s) out[#out+1] = s end
if nan < 5 then p("lt") else p("-") end
if nan >= 5 then p("ge") else p("-") end
if nan <= 5 then p("le") else p("-") end
if nan > 5 then p("gt") else p("-") end
if nan == nan then p("eq") else p("-") end
if nan ~= nan then p("ne") else p("-") end
local five = 5
if nan < five then p("lt") else p("-") end
if nan >= five then p("ge") else p("-") end
if nan <= five then p("le") else p("-") end
if nan > five then p("gt") else p("-") end
while nan < 5 do p("loop") break end
repeat p("once") until nan ~= nan
return table.concat(out, " ")
`
	want := "- - - - - ne - - - - once"
	if got := firstString(t, runSrcOK(t, src)); got != want {
		t.Errorf("NaN semantics mismatch\n got: %s\nwant: %s", got, want)
	}
}

// TestFusedConditionsLiveOperands guards the register-allocation invariant the
// fused path depends on: the compare reads its operands in place, so a local
// that a metamethod or upvalue write mutates must be read live.
func TestFusedConditionsLiveOperands(t *testing.T) {
	src := `
local out = {}
local n = 1
local function bump() n = 100 return 2 end
if n < bump() then out[#out+1] = "lt" else out[#out+1] = "-" end
out[#out+1] = tostring(n)
local m = 1
local function bump2() m = 100 return 2 end
if bump2() < m then out[#out+1] = "lt2" else out[#out+1] = "-" end
return table.concat(out, " ")
`
	// lua5.5.0: n is read AFTER bump() runs (both operands are on the stack
	// when LT executes, but n lives in a captured upvalue cell), so n<2 is
	// false; the second case compares 2 < 100 -> true.
	want := "- 100 lt2"
	if got := firstString(t, runSrcOK(t, src)); got != want {
		t.Errorf("live operand mismatch\n got: %s\nwant: %s", got, want)
	}
}

// TestRepeatUntilClosesUpvalues covers the restructured repeat/until close
// stub: the fused test and its JMP must stay adjacent, so OP_CLOSE moved onto
// the two branch paths. Each iteration must still get a fresh upvalue cell,
// and a to-be-closed variable must still fire exactly once per iteration.
func TestRepeatUntilClosesUpvalues(t *testing.T) {
	src := `
local fns = {}
local i = 0
repeat
  i = i + 1
  local x = i
  fns[#fns+1] = function() return x end
until i >= 3
local out = {}
for k = 1, #fns do out[#out+1] = tostring(fns[k]()) end
return table.concat(out, " ")
`
	if got := firstString(t, runSrcOK(t, src)); got != "1 2 3" {
		t.Errorf("repeat upvalue-per-iteration mismatch: got %q, want %q", got, "1 2 3")
	}

	tbc := `
local out = {}
local i = 0
repeat
  i = i + 1
  local x <close> = setmetatable({}, {__close = function() out[#out+1] = "close" .. i end})
until i >= 3
return table.concat(out, " ")
`
	if got := firstString(t, runSrcOK(t, tbc)); got != "close1 close2 close3" {
		t.Errorf("repeat <close> mismatch: got %q, want %q", got, "close1 close2 close3")
	}

	// The until-condition itself may capture a body local, which is what makes
	// the close decision only knowable after the condition is compiled.
	late := `
local i = 0
local seen = {}
repeat
  i = i + 1
  local x = i
  seen[#seen+1] = x
until (function() return x >= 3 end)()
return #seen .. ":" .. table.concat(seen, ",")
`
	if got := firstString(t, runSrcOK(t, late)); got != "3:1,2,3" {
		t.Errorf("repeat late-capture mismatch: got %q, want %q", got, "3:1,2,3")
	}
}

// TestFusedControlFlowResults exercises the fused shapes end-to-end so a wrong
// k sense or a mis-signed jump shows up as a wrong answer rather than a wrong
// disassembly.
func TestFusedControlFlowResults(t *testing.T) {
	src := `
local out = {}
local function p(x) out[#out+1] = tostring(x) end

-- if / elseif / else with every operator, both operand orders
for _, v in ipairs{-1, 0, 1, 5, 6} do
  if v < 0 then p("neg")
  elseif v == 0 then p("zero")
  elseif v <= 5 then p("small")
  else p("big") end
end

-- while with an immediate bound
local n = 0
while n < 4 do n = n + 1 end
p(n)

-- while with a register bound
local lim = 7
local m = 0
while m < lim do m = m + 1 end
p(m)

-- repeat with an immediate bound
local r = 10
repeat r = r - 1 until r <= 6
p(r)

-- not / parentheses
local flag = false
if not flag then p("notfalse") end
if not (1 > 2) then p("notparen") end
if (3 == 3) then p("paren") end

-- nested and/or still short-circuit
local calls = 0
local function side() calls = calls + 1 return true end
if 1 < 2 and side() then p("and") end
if 1 > 2 and side() then p("BAD") end
if 1 > 2 or side() then p("or") end
p(calls)

-- reversed constants
local z = 3
if 5 > z then p("rev-gt") end
if 5 >= z then p("rev-ge") end
if 0 < z then p("rev-lt") end
if 3 == z then p("rev-eq") end
if 4 ~= z then p("rev-ne") end
return table.concat(out, " ")
`
	want := "neg zero small small big 4 7 6 notfalse notparen paren and or 2 rev-gt rev-ge rev-lt rev-eq rev-ne"
	if got := firstString(t, runSrcOK(t, src)); got != want {
		t.Errorf("control flow mismatch\n got: %s\nwant: %s", got, want)
	}
}

// TestFusedConditionEdgeShapes covers the transparent-parenthesis and
// negation rewrites: a parenthesised multi-value call must still truncate to
// one value, `not` must only flip the jump sense, and `and`/`or` conditions
// must keep short-circuiting.
func TestFusedConditionEdgeShapes(t *testing.T) {
	src := `
local out = {}
local function p(x) out[#out+1] = tostring(x) end

local function multi() return nil, true end
local function one() return 7 end

if (multi()) then p("m-y") else p("m-n") end
if multi() then p("m2-y") else p("m2-n") end
if (one()) then p("o-y") else p("o-n") end
if not (multi()) then p("nm-y") else p("nm-n") end

local function va(...)
  if (...) then p("va-y") else p("va-n") end
  if not (...) then p("nva-y") else p("nva-n") end
  if select("#", ...) > 1 then p("many") else p("few") end
end
va(true, false)
va(false, true)
va()

local x = false
if not not x then p("nn-y") else p("nn-n") end
x = 0
if not not x then p("nn2-y") else p("nn2-n") end

local a, b = 1, 2
if not (a < b) then p("nab-y") else p("nab-n") end
if not (a > b) then p("nab2-y") else p("nab2-n") end
if not (a < b and b < 3) then p("nand-y") else p("nand-n") end
if not (a > b or b > 3) then p("nor-y") else p("nor-n") end

local flag = nil
if (flag) then p("f-y") else p("f-n") end
while (a < 4) do a = a + 1 end
p(a)
local c = 0
repeat c = c + 1 until (c >= 3)
p(c)
local d = 0
repeat d = d + 1 until not (d < 3)
p(d)

local hits = 0
local function eff() hits = hits + 1 return 5 end
if eff() < 10 then p("eff-y") end
if 10 < eff() then p("BAD") else p("eff-n") end
if eff() == 5 then p("eff-eq") end
if eff() ~= 5 then p("BAD2") else p("eff-ne") end
p(hits)

local v1 = a < b
local v2 = a < 5
local v3 = 5 < a
local v4 = a == 4
local v5 = a == "z"
p(tostring(v1) .. tostring(v2) .. tostring(v3) .. tostring(v4) .. tostring(v5))
return table.concat(out, " ")
`
	// Reference output from lua5.5.0 on identical source.
	want := "m-n m2-n o-y nm-y va-y nva-n many va-n nva-y many va-n nva-y few " +
		"nn-n nn2-y nab-n nab2-y nand-n nor-y f-n 4 3 3 " +
		"eff-y eff-n eff-eq eff-ne 4 falsetruefalsetruefalse"
	if got := firstString(t, runSrcOK(t, src)); got != want {
		t.Errorf("edge shape mismatch\n got: %s\nwant: %s", got, want)
	}
}

// TestFusedComparisonLineAttribution pins the source line the fused compare is
// attributed to: reference Lua reports a comparison error at the line of the
// comparison's LAST token, not the `if`/`then` line.
func TestFusedComparisonLineAttribution(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{"local t = {}\nif t < 5 then end", "test:2:"},
		{"local t = {}\nif t\n  <\n  5 then end", "test:4:"},
		{"local t = {}\nwhile t <= 5 do end", "test:2:"},
		{"local t = {}\nrepeat until t >= 5", "test:2:"},
	}
	for _, tc := range cases {
		_, err := runSrc(t, tc.src)
		if err == nil {
			t.Errorf("%q: expected an error", tc.src)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%q:\n got: %v\nwant prefix: %s", tc.src, err, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Part 4: hook-visible instruction counts.
//
// These are deterministic and load-independent, and are the measurement the
// optimization is judged on: a count hook must not see GoLua executing far
// more instructions than reference Lua for the same source.
// ---------------------------------------------------------------------------

func TestCountHookInstructionCounts(t *testing.T) {
	cases := []struct {
		name string
		body string
		want int // GoLua count after fusion
		ref  int // lua5.5.0 count for the same source
		was  int // GoLua count before fusion
	}{
		// Totals a count-1 hook observes between sethook/sethook(). The `ref`
		// column is /usr/bin/lua5.5.0 on identical source; it is documentation,
		// not an assertion, because a few unrelated shape differences remain
		// (see tests/doctest/compiler_count_hook_shapes.lua).
		{"if immediate", "local d = 0\nif d == 0 then d = 1 end", 6, 6, 10},
		{"while immediate", "local x = 0\nwhile x < 4 do x = x + 1 end", 22, 17, 42},
		{"repeat immediate", "local x = 4\nrepeat x = x - 1 until x < 1", 19, 12, 35},
	}
	for _, tc := range cases {
		src := "local n = 0\ndebug.sethook(function() n = n + 1 end, '', 1)\n" +
			tc.body + "\ndebug.sethook()\nreturn n"
		res := runSrcOK(t, src)
		got := res[0].String()
		if got != strconv.Itoa(tc.want) {
			t.Errorf("%s: count hook saw %s instructions, want %d (lua5.5.0: %d, before fusion: %d)",
				tc.name, got, tc.want, tc.ref, tc.was)
		}
	}
}
