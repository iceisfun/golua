package compiler

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/parser"
)

// compile is a test helper that parses+compiles Lua source.
func compile(t *testing.T, source string) *Proto {
	t.Helper()
	block, err := parser.Parse("<test>", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := Compile("<test>", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	return proto
}

// hasOp checks that the proto contains at least one instruction with the given opcode.
func hasOp(p *Proto, op OpCode) bool {
	for _, inst := range p.Code {
		if inst.OpCode() == op {
			return true
		}
	}
	return false
}

// countOp counts instructions with the given opcode.
func countOp(p *Proto, op OpCode) int {
	n := 0
	for _, inst := range p.Code {
		if inst.OpCode() == op {
			n++
		}
	}
	return n
}

func TestEmptyChunk(t *testing.T) {
	p := compile(t, "")
	if !hasOp(p, OP_VARARGPREP) {
		t.Error("missing VARARGPREP")
	}
	if !hasOp(p, OP_RETURN0) {
		t.Error("missing RETURN0")
	}
}

func TestReturnLiteral(t *testing.T) {
	p := compile(t, "return 42")
	if !hasOp(p, OP_LOADI) {
		t.Error("expected LOADI for small integer")
	}
	if !hasOp(p, OP_RETURN1) {
		t.Error("expected RETURN1 for single-value return")
	}
}

func TestReturnNil(t *testing.T) {
	p := compile(t, "return nil")
	if !hasOp(p, OP_LOADNIL) {
		t.Error("expected LOADNIL")
	}
}

func TestReturnBooleans(t *testing.T) {
	p := compile(t, "return true, false")
	if !hasOp(p, OP_LOADTRUE) {
		t.Error("expected LOADTRUE")
	}
	if !hasOp(p, OP_LOADFALSE) {
		t.Error("expected LOADFALSE")
	}
	if !hasOp(p, OP_RETURN) {
		t.Error("expected RETURN (multi-value)")
	}
}

func TestReturnString(t *testing.T) {
	p := compile(t, `return "hello"`)
	if !hasOp(p, OP_LOADK) {
		t.Error("expected LOADK for string constant")
	}
	if len(p.Constants) == 0 {
		t.Fatal("expected at least one constant")
	}
	found := false
	for _, k := range p.Constants {
		if k.Type == ValString && k.SVal == "hello" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected string constant 'hello'")
	}
}

func TestReturnFloat(t *testing.T) {
	p := compile(t, "return 3.14")
	if !hasOp(p, OP_LOADK) {
		t.Error("expected LOADK for float constant")
	}
}

func TestReturnIntegerFloat(t *testing.T) {
	// Float with integer value should use LOADF
	p := compile(t, "return 5.0")
	if !hasOp(p, OP_LOADF) {
		t.Error("expected LOADF for integer-valued float")
	}
}

func TestLargeInteger(t *testing.T) {
	p := compile(t, "return 1000000")
	// 1000000 > OffsetSBx(65535), so should use LOADK
	if !hasOp(p, OP_LOADK) {
		t.Error("expected LOADK for large integer")
	}
}

func TestLocalVariable(t *testing.T) {
	p := compile(t, "local x = 10; return x")
	if !hasOp(p, OP_LOADI) {
		t.Error("expected LOADI")
	}
	if !hasOp(p, OP_RETURN1) {
		t.Error("expected RETURN1")
	}
}

func TestLocalMultiple(t *testing.T) {
	p := compile(t, "local a, b = 1, 2; return a + b")
	if !hasOp(p, OP_ADD) {
		t.Error("expected ADD")
	}
}

func TestGlobalAccess(t *testing.T) {
	p := compile(t, "return x")
	if !hasOp(p, OP_GETTABUP) {
		t.Error("expected GETTABUP for global access")
	}
}

func TestGlobalAssign(t *testing.T) {
	p := compile(t, "x = 42")
	if !hasOp(p, OP_SETTABUP) {
		t.Error("expected SETTABUP for global assignment")
	}
}

func TestArithmeticOps(t *testing.T) {
	tests := []struct {
		src string
		op  OpCode
	}{
		{"local a,b = 1,2; return a+b", OP_ADD},
		{"local a,b = 1,2; return a-b", OP_SUB},
		{"local a,b = 1,2; return a*b", OP_MUL},
		{"local a,b = 1,2; return a/b", OP_DIV},
		{"local a,b = 1,2; return a//b", OP_IDIV},
		{"local a,b = 1,2; return a%b", OP_MOD},
		{"local a,b = 1,2; return a^b", OP_POW},
		{"local a,b = 1,2; return a&b", OP_BAND},
		{"local a,b = 1,2; return a|b", OP_BOR},
		{"local a,b = 1,2; return a~b", OP_BXOR},
		{"local a,b = 1,2; return a<<b", OP_SHL},
		{"local a,b = 1,2; return a>>b", OP_SHR},
	}
	for _, tt := range tests {
		t.Run(OpName(tt.op), func(t *testing.T) {
			p := compile(t, tt.src)
			if !hasOp(p, tt.op) {
				t.Errorf("expected %s in bytecode", OpName(tt.op))
			}
		})
	}
}

func TestUnaryOps(t *testing.T) {
	tests := []struct {
		src string
		op  OpCode
	}{
		{"local a = 1; return -a", OP_UNM},
		{"local a = true; return not a", OP_NOT},
		{"local a = 'hi'; return #a", OP_LEN},
		{"local a = 7; return ~a", OP_BNOT},
	}
	for _, tt := range tests {
		t.Run(OpName(tt.op), func(t *testing.T) {
			p := compile(t, tt.src)
			if !hasOp(p, tt.op) {
				t.Errorf("expected %s", OpName(tt.op))
			}
		})
	}
}

func TestConcat(t *testing.T) {
	p := compile(t, `local a,b,c = "x","y","z"; return a..b..c`)
	if !hasOp(p, OP_CONCAT) {
		t.Error("expected CONCAT")
	}
}

func TestIfStmt(t *testing.T) {
	p := compile(t, `
		local x = 10
		if x then
			return 1
		else
			return 2
		end
	`)
	if !hasOp(p, OP_TEST) {
		t.Error("expected TEST for if condition")
	}
	if !hasOp(p, OP_JMP) {
		t.Error("expected JMP")
	}
}

func TestWhileLoop(t *testing.T) {
	p := compile(t, `
		local i = 0
		while i do
			i = i + 1
		end
	`)
	if !hasOp(p, OP_TEST) {
		t.Error("expected TEST")
	}
	jmpCount := countOp(p, OP_JMP)
	if jmpCount < 2 {
		t.Errorf("expected at least 2 JMPs (got %d)", jmpCount)
	}
}

func TestRepeatLoop(t *testing.T) {
	p := compile(t, `
		local x = 0
		repeat
			x = x + 1
		until x
	`)
	if !hasOp(p, OP_TEST) {
		t.Error("expected TEST")
	}
}

func TestForNumLoop(t *testing.T) {
	p := compile(t, "for i = 1, 10 do end")
	if !hasOp(p, OP_FORPREP) {
		t.Error("expected FORPREP")
	}
	if !hasOp(p, OP_FORLOOP) {
		t.Error("expected FORLOOP")
	}
}

func TestForNumWithStep(t *testing.T) {
	p := compile(t, "for i = 1, 10, 2 do end")
	if !hasOp(p, OP_FORPREP) {
		t.Error("expected FORPREP")
	}
}

func TestForInLoop(t *testing.T) {
	p := compile(t, "for k, v in pairs({}) do end")
	if !hasOp(p, OP_TFORPREP) {
		t.Error("expected TFORPREP")
	}
	if !hasOp(p, OP_TFORCALL) {
		t.Error("expected TFORCALL")
	}
	if !hasOp(p, OP_TFORLOOP) {
		t.Error("expected TFORLOOP")
	}
}

func TestFuncCall(t *testing.T) {
	p := compile(t, "print(42)")
	if !hasOp(p, OP_CALL) {
		t.Error("expected CALL")
	}
	if !hasOp(p, OP_GETTABUP) {
		t.Error("expected GETTABUP for global 'print'")
	}
}

func TestMethodCall(t *testing.T) {
	p := compile(t, "local t = {}; t:foo(1)")
	if !hasOp(p, OP_SELF) {
		t.Error("expected SELF")
	}
	if !hasOp(p, OP_CALL) {
		t.Error("expected CALL")
	}
}

func TestFuncDef(t *testing.T) {
	p := compile(t, `
		local function add(a, b)
			return a + b
		end
	`)
	if !hasOp(p, OP_CLOSURE) {
		t.Error("expected CLOSURE")
	}
	if len(p.Protos) != 1 {
		t.Fatalf("expected 1 sub-proto, got %d", len(p.Protos))
	}
	sub := p.Protos[0]
	if sub.NumParams != 2 {
		t.Errorf("expected 2 params, got %d", sub.NumParams)
	}
	if !hasOp(sub, OP_ADD) {
		t.Error("sub-proto should have ADD")
	}
}

func TestReturnEmpty(t *testing.T) {
	p := compile(t, "return")
	if !hasOp(p, OP_RETURN0) {
		t.Error("expected RETURN0")
	}
}

func TestTailCall(t *testing.T) {
	p := compile(t, `
		local function f() end
		return f()
	`)
	if !hasOp(p, OP_TAILCALL) {
		t.Error("expected TAILCALL")
	}
}

func TestTableConstructorEmpty(t *testing.T) {
	p := compile(t, "local t = {}")
	if !hasOp(p, OP_NEWTABLE) {
		t.Error("expected NEWTABLE")
	}
}

func TestTableConstructorArray(t *testing.T) {
	p := compile(t, "local t = {1, 2, 3}")
	if !hasOp(p, OP_NEWTABLE) {
		t.Error("expected NEWTABLE")
	}
	if !hasOp(p, OP_SETLIST) {
		t.Error("expected SETLIST")
	}
}

func TestTableConstructorRecord(t *testing.T) {
	p := compile(t, `local t = {x = 1, y = 2}`)
	if !hasOp(p, OP_NEWTABLE) {
		t.Error("expected NEWTABLE")
	}
	if !hasOp(p, OP_SETFIELD) {
		t.Error("expected SETFIELD")
	}
}

func TestFieldAccess(t *testing.T) {
	p := compile(t, "local t = {}; return t.x")
	if !hasOp(p, OP_GETFIELD) {
		t.Error("expected GETFIELD")
	}
}

func TestFieldAssign(t *testing.T) {
	p := compile(t, "local t = {}; t.x = 42")
	if !hasOp(p, OP_SETFIELD) {
		t.Error("expected SETFIELD")
	}
}

func TestIndexAccess(t *testing.T) {
	p := compile(t, "local t = {}; local k = 1; return t[k]")
	if !hasOp(p, OP_GETTABLE) {
		t.Error("expected GETTABLE")
	}
}

func TestDoBlock(t *testing.T) {
	// Just verify it compiles and scoping works
	p := compile(t, `
		local x = 1
		do
			local y = 2
			x = y
		end
		return x
	`)
	_ = p
}

func TestBreak(t *testing.T) {
	p := compile(t, `
		while true do
			break
		end
	`)
	if !hasOp(p, OP_JMP) {
		t.Error("expected JMP for break")
	}
}

func TestGotoLabel(t *testing.T) {
	p := compile(t, `
		goto skip
		print("unreachable")
		::skip::
		return 2
	`)
	if !hasOp(p, OP_JMP) {
		t.Error("expected JMP for goto")
	}
}

func TestGotoIntoScopeError(t *testing.T) {
	block, err := parser.Parse("<test>", `
		goto skip
		local x = 1
		::skip::
		return x
	`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	_, err = Compile("<test>", block)
	if err == nil {
		t.Error("expected compile error for goto jumping into local scope")
	}
}

func TestLogicalAnd(t *testing.T) {
	// Result to a temporary (return slot): short-circuits with TEST in place,
	// reusing one register (so chains don't exhaust registers).
	p := compile(t, "local a, b = 1, 2; return a and b")
	if !hasOp(p, OP_TEST) {
		t.Error("expected TEST for 'and' to a temporary")
	}
	// Result to a live local: uses TESTSET via a fresh temp so the local isn't
	// clobbered before the right operand reads it.
	p2 := compile(t, "local a, b = 1, 2; a = a and b; return a")
	if !hasOp(p2, OP_TESTSET) {
		t.Error("expected TESTSET for 'and' into a live local")
	}
}

func TestLogicalOr(t *testing.T) {
	p := compile(t, "local a, b = 1, 2; return a or b")
	if !hasOp(p, OP_TEST) {
		t.Error("expected TEST for 'or' to a temporary")
	}
	p2 := compile(t, "local a, b = 1, 2; a = a or b; return a")
	if !hasOp(p2, OP_TESTSET) {
		t.Error("expected TESTSET for 'or' into a live local")
	}
}

func TestComparison(t *testing.T) {
	tests := []struct {
		src string
		op  OpCode
	}{
		{"local a,b = 1,2; return a == b", OP_EQ},
		{"local a,b = 1,2; return a ~= b", OP_EQ},
		{"local a,b = 1,2; return a < b", OP_LT},
		{"local a,b = 1,2; return a <= b", OP_LE},
		{"local a,b = 1,2; return a > b", OP_LT},
		{"local a,b = 1,2; return a >= b", OP_LE},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			p := compile(t, tt.src)
			if !hasOp(p, tt.op) {
				t.Errorf("expected %s", OpName(tt.op))
			}
		})
	}
}

func TestMultiAssign(t *testing.T) {
	p := compile(t, "local a, b; a, b = 1, 2")
	if countOp(p, OP_LOADI) < 2 {
		t.Error("expected at least 2 LOADI")
	}
}

func TestVararg(t *testing.T) {
	// Top-level chunk IS vararg
	p := compile(t, "return ...")
	if !hasOp(p, OP_VARARG) {
		t.Error("expected VARARG")
	}
}

func TestVarargFunction(t *testing.T) {
	p := compile(t, `
		local function f(...)
			return ...
		end
	`)
	if len(p.Protos) != 1 {
		t.Fatal("expected 1 sub-proto")
	}
	sub := p.Protos[0]
	if !sub.IsVarArg {
		t.Error("sub-proto should be vararg")
	}
	if !hasOp(sub, OP_VARARGPREP) {
		t.Error("expected VARARGPREP")
	}
	if !hasOp(sub, OP_VARARG) {
		t.Error("expected VARARG")
	}
}

func TestGlobalFunc(t *testing.T) {
	p := compile(t, `
		function foo(x)
			return x + 1
		end
	`)
	if !hasOp(p, OP_CLOSURE) {
		t.Error("expected CLOSURE")
	}
	if !hasOp(p, OP_SETTABUP) {
		t.Error("expected SETTABUP (global function)")
	}
}

func TestNestedFunction(t *testing.T) {
	p := compile(t, `
		local function outer()
			local x = 10
			local function inner()
				return x
			end
			return inner
		end
	`)
	if len(p.Protos) != 1 {
		t.Fatal("expected 1 top-level sub-proto")
	}
	outer := p.Protos[0]
	if len(outer.Protos) != 1 {
		t.Fatal("expected 1 nested sub-proto in outer")
	}
	inner := outer.Protos[0]
	// inner should have an upvalue for 'x'
	if len(inner.Upvalues) < 1 {
		t.Error("inner should have upvalues")
	}
}

func TestElseIf(t *testing.T) {
	p := compile(t, `
		local x = 1
		if x == 1 then
			return 'one'
		elseif x == 2 then
			return 'two'
		else
			return 'other'
		end
	`)
	// Should have multiple TEST/JMP sequences
	if countOp(p, OP_JMP) < 2 {
		t.Error("expected multiple JMPs for if/elseif/else")
	}
}

func TestDump(t *testing.T) {
	p := compile(t, `
		local x = 42
		print(x)
		return x + 1
	`)
	dump := p.DumpString()
	if !strings.Contains(dump, "LOADI") {
		t.Error("dump should contain LOADI")
	}
	if !strings.Contains(dump, "CALL") {
		t.Error("dump should contain CALL")
	}
	if !strings.Contains(dump, "constants") {
		t.Error("dump should contain constants section")
	}
}

func TestInstructionEncoding(t *testing.T) {
	// Test round-trip encoding/decoding
	inst := ABC(OP_ADD, 3, 1, 2, 0)
	if inst.OpCode() != OP_ADD {
		t.Errorf("opcode: got %d, want %d", inst.OpCode(), OP_ADD)
	}
	if inst.A() != 3 {
		t.Errorf("A: got %d, want 3", inst.A())
	}
	if inst.B() != 1 {
		t.Errorf("B: got %d, want 1", inst.B())
	}
	if inst.C() != 2 {
		t.Errorf("C: got %d, want 2", inst.C())
	}

	inst2 := AsBx(OP_LOADI, 5, -10)
	if inst2.OpCode() != OP_LOADI {
		t.Errorf("opcode: got %d, want %d", inst2.OpCode(), OP_LOADI)
	}
	if inst2.A() != 5 {
		t.Errorf("A: got %d, want 5", inst2.A())
	}
	if inst2.SBx() != -10 {
		t.Errorf("SBx: got %d, want -10", inst2.SBx())
	}

	inst3 := SJ(OP_JMP, -5, 0)
	if inst3.OpCode() != OP_JMP {
		t.Errorf("opcode: got %d, want %d", inst3.OpCode(), OP_JMP)
	}
	if inst3.SJ() != -5 {
		t.Errorf("SJ: got %d, want -5", inst3.SJ())
	}
}

func TestFibonacci(t *testing.T) {
	p := compile(t, `
		local function fib(n)
			if n < 2 then
				return n
			end
			return fib(n-1) + fib(n-2)
		end
		return fib(10)
	`)
	if len(p.Protos) != 1 {
		t.Fatal("expected 1 sub-proto for fib")
	}
	fib := p.Protos[0]
	if fib.NumParams != 1 {
		t.Errorf("fib should have 1 param, got %d", fib.NumParams)
	}
	// Lua 5.5 rewrites x - <int_literal> as ADDI with negated immediate, so
	// fib(n-1) / fib(n-2) emit OP_ADDI (not OP_SUB). The outer fib(...)+fib(...)
	// remains a register-register OP_ADD.
	if !hasOp(fib, OP_LT) || !hasOp(fib, OP_ADDI) || !hasOp(fib, OP_ADD) {
		t.Error("fib should have LT, ADDI, ADD")
	}
}

func TestConstantDedup(t *testing.T) {
	p := compile(t, `return "hello", "world", "hello"`)
	// "hello" should appear only once in constants
	count := 0
	for _, k := range p.Constants {
		if k.Type == ValString && k.SVal == "hello" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 'hello' to appear once in constants, got %d", count)
	}
}

func TestParenExpr(t *testing.T) {
	// (expr) should truncate multi-return to single value
	p := compile(t, "local function f() return 1, 2 end; return (f())")
	// Should compile without error
	_ = p
}

// TestAssignCallRegAlloc verifies that assigning a function call result to an
// existing local uses the first free register for the call (matching PUC-Lua).
// Before the fix, compileSingleAssign pre-reserved a temp register, inflating
// freeReg so that compileExprToReg allocated a *second* temp, producing an
// extra OP_MOVE and wasting a register.
//
// PUC-Lua bytecode for "local x = 1; x = print(x)":
//
//	VARARGPREP 0; LOADI 0 1; GETTABUP 1 0 0; MOVE 2 0; CALL 1 2 2; MOVE 0 1; RETURN 1 1 1
//
// (7 instructions, 2 OP_MOVE: one for arg copy, one for final assignment)
//
// GoLua before fix (8 instructions, 3 OP_MOVE):
//
//	VARARGPREP 0; LOADI 0 1; GETTABUP 2 0 0; MOVE 3 0; CALL 2 2 2; MOVE 1 2; MOVE 0 1; RETURN0
func TestAssignCallRegAlloc(t *testing.T) {
	p := compile(t, `local x = 1; x = print(x)`)
	moves := countOp(p, OP_MOVE)
	// 2 MOVEs: MOVE for arg copy + MOVE for final assignment (matches PUC-Lua).
	if moves != 2 {
		t.Errorf("expected 2 OP_MOVE, got %d\nbytecode:\n%s", moves, p.DumpString())
	}
	// Total instruction count should match PUC-Lua (7).
	if len(p.Code) > 7 {
		t.Errorf("expected <= 7 instructions, got %d\nbytecode:\n%s", len(p.Code), p.DumpString())
	}
}

// TestAssignMethodCallRegAlloc verifies method call assignment register usage.
// PUC-Lua produces 6 instructions (SELF reads directly from the local).
// GoLua currently produces 7 (extra MOVE to copy object before SELF) — the
// compileMethodCall object-copy is a separate issue from the assignment path.
func TestAssignMethodCallRegAlloc(t *testing.T) {
	p := compile(t, `local s = "hello"; s = s:upper()`)
	moves := countOp(p, OP_MOVE)
	// 2 MOVEs: MOVE for object copy before SELF + MOVE for final assignment.
	// (PUC-Lua only needs 1 MOVE since SELF reads the object register directly.)
	if moves != 2 {
		t.Errorf("expected 2 OP_MOVE, got %d\nbytecode:\n%s", moves, p.DumpString())
	}
	// 7 instructions (one more than PUC-Lua's 6 due to object copy).
	if len(p.Code) > 7 {
		t.Errorf("expected <= 7 instructions, got %d\nbytecode:\n%s", len(p.Code), p.DumpString())
	}
}

// TestControlStructureTooLong_ForNum verifies that the compiler rejects a
// numeric for loop whose body exceeds the maximum unsigned Bx field (17-bit,
// MaxArgBx = 131071). Each "a=1;" compiles to 1 instruction (LOADI into a
// local register), so MaxArgBx+1 repetitions produces a bodyLen that overflows.
func TestControlStructureTooLong_ForNum(t *testing.T) {
	// Build a for-loop whose body has exactly MaxArgBx+1 instructions.
	// "local a\n for i=1,1 do <body> end"
	// where body = "a=1\n" repeated (MaxArgBx+1) times.
	// Each "a=1" into an outer local is 1 LOADI instruction.
	n := MaxArgBx + 1 // 131072
	body := strings.Repeat("a=1\n", n)
	src := "local a\nfor i=1,1 do\n" + body + "end\n"

	block, err := parser.Parse("<test>", src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	_, err = Compile("<test>", block)
	if err == nil {
		t.Fatal("expected compile error for control structure too long")
	}
	if !strings.Contains(err.Error(), "too long") {
		t.Fatalf("expected 'too long' in error, got: %v", err)
	}
}

// TestControlStructureTooLong_ForNumJustFits verifies that a numeric for loop
// with exactly MaxArgBx body instructions compiles successfully.
func TestControlStructureTooLong_ForNumJustFits(t *testing.T) {
	n := MaxArgBx // 131071 -- exactly at the limit
	body := strings.Repeat("a=1\n", n)
	src := "local a\nfor i=1,1 do\n" + body + "end\n"

	block, err := parser.Parse("<test>", src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	_, err = Compile("<test>", block)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
}

// TestAllLuaTestFiles tries to compile all the Lua test files that the parser handles.
func TestCompileLuaTestFiles(t *testing.T) {
	// These are the test files from the Lua 5.5 test suite that the parser can handle.
	// We just verify they compile without error (semantic correctness tested in VM session).
	sources := []struct {
		name string
		src  string
	}{
		{"assignment", "local a = 1; a = a + 1"},
		{"multi-return", "local function f() return 1, 2, 3 end"},
		{"table-ops", "local t = {1, 2, x=3}; t[1] = 4; t.y = 5"},
		{"nested-if", "local x = 1; if x > 0 then if x > 1 then return 2 end; return 1 end"},
		{"for-numeric", "local s = 0; for i = 1, 100 do s = s + i end; return s"},
		{"closures", `
			local function make()
				local x = 0
				return function() x = x + 1; return x end
			end
		`},
	}

	for _, tt := range sources {
		t.Run(tt.name, func(t *testing.T) {
			compile(t, tt.src)
		})
	}
}

// TestConstantFoldNegativeZero verifies that constant folding does not
// freeze the sign of negative zero (-0.0) into a single compile-time
// constant. Reference Lua's constfolding() declines to fold when the
// float result is NaN or 0, leaving the runtime computation to produce
// the correct IEEE sign of zero. We verify by ensuring NO compile-time
// constant equals the result for these expressions — the work must
// happen at runtime via UNM/ADD/SUB/MUL etc.
func TestConstantFoldNegativeZero(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"mul_0_neg1", "return 0.0 * -1"},
		{"add_neg0_neg0", "return -0.0 + -0.0"},
		{"sub_0_0", "return 0.0 - 0.0"},
		{"mul_neg0_1", "return -0.0 * 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := compile(t, tt.src)
			// No folded zero constant should be in the constant table — that
			// would mean the compiler folded the result and lost the sign
			// distinction. (UNM at runtime preserves the correct sign.)
			for _, k := range p.Constants {
				if k.Type == ValFloat && k.FVal == 0 {
					t.Errorf("expression %q: zero float constant %v in pool — folder should leave it to runtime", tt.name, k.FVal)
				}
			}
			// At least one arithmetic op must remain (no fold == runtime work).
			hasArith := false
			for _, inst := range p.Code {
				switch inst.OpCode() {
				case OP_ADD, OP_SUB, OP_MUL, OP_DIV, OP_MOD, OP_IDIV, OP_POW,
					OP_ADDI, OP_ADDK, OP_SUBK, OP_MULK, OP_DIVK, OP_MODK, OP_POWK, OP_IDIVK,
					OP_UNM:
					hasArith = true
				}
			}
			if !hasArith {
				t.Errorf("expression %q: expected runtime arithmetic op (no fold), but none found", tt.name)
			}
		})
	}
}

// TestGotoIntoScopeErrorLine_Repeat verifies that goto-into-scope errors
// in repeat...until blocks report the line of the until keyword, not the
// repeat keyword.
func TestGotoIntoScopeErrorLine_Repeat(t *testing.T) {
	src := "repeat\ngoto L\nlocal x = 1\n::L::\nuntil x"
	block, err := parser.Parse("<test>", src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	// Do NOT pass WithEndLine — this tests that the compiler computes the
	// correct line from the AST alone (blockMaxLine fallback).
	_, err = Compile("<test>", block)
	if err == nil {
		t.Fatal("expected compile error for goto jumping into local scope")
	}
	errMsg := err.Error()
	// Should report line 5 (until keyword), not line 1 (repeat keyword)
	if !strings.Contains(errMsg, "]:5:") {
		t.Errorf("expected error at line 5 (until), got: %s", errMsg)
	}
}

// TestGotoIntoScopeErrorLine_While verifies that goto-into-scope errors
// in while/do blocks report the line of the first statement after the
// label, not the line of the end keyword.
func TestGotoIntoScopeErrorLine_While(t *testing.T) {
	src := "while true do\ngoto L\nlocal x\n::L::\nprint(1)\nbreak\nend"
	block, err := parser.Parse("<test>", src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	// Do NOT pass WithEndLine — this tests that the compiler computes the
	// correct line from the AST alone.
	_, err = Compile("<test>", block)
	if err == nil {
		t.Fatal("expected compile error for goto jumping into local scope")
	}
	errMsg := err.Error()
	// Should report line 5 (print(1), the statement after ::L::), not line 7 (end)
	if !strings.Contains(errMsg, "]:5:") {
		t.Errorf("expected error at line 5 (first stmt after label), got: %s", errMsg)
	}
}
