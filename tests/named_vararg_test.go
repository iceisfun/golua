package tests

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
)

func TestNamedVarArgBasic(t *testing.T) {
	_, lines, err := runLua(t, `
		function f(... args)
			print(type(args), args.n, args[1], args[2])
		end
		f(10, 20)
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 1 || lines[0] != "table\t2\t10\t20" {
		t.Fatalf("expected 'table\\t2\\t10\\t20', got %v", lines)
	}
}

func TestNamedVarArgWithFixedParams(t *testing.T) {
	_, lines, err := runLua(t, `
		function f(a, b, ... rest)
			print(a, b, rest.n, rest[1], rest[2])
		end
		f(1, 2, 3, 4)
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 1 || lines[0] != "1\t2\t2\t3\t4" {
		t.Fatalf("expected '1\\t2\\t2\\t3\\t4', got %v", lines)
	}
}

func TestNamedVarArgEmpty(t *testing.T) {
	_, lines, err := runLua(t, `
		function f(... args)
			print(args.n)
		end
		f()
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 1 || lines[0] != "0" {
		t.Fatalf("expected '0', got %v", lines)
	}
}

func TestNamedVarArgWithNils(t *testing.T) {
	_, lines, err := runLua(t, `
		function f(... args)
			print(args.n, args[1], args[2], args[3])
		end
		f(nil, 42, nil)
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 1 || lines[0] != "3\tnil\t42\tnil" {
		t.Fatalf("expected '3\\tnil\\t42\\tnil', got %v", lines)
	}
}

func TestNamedVarArgDotsStillWork(t *testing.T) {
	_, lines, err := runLua(t, `
		function f(... args)
			print(...)
			print(args[1])
		end
		f(1, 2, 3)
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 2 || lines[0] != "1\t2\t3" || lines[1] != "1" {
		t.Fatalf("expected ['1\\t2\\t3', '1'], got %v", lines)
	}
}

func TestNamedVarArgIsConst(t *testing.T) {
	// Named vararg parameter should be const — assignment should fail at compile time
	code := `return function(... t) t = 10 end`
	block, err := parser.Parse("test", code)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	_, compileErr := compiler.Compile("test", block)
	if compileErr == nil {
		t.Fatalf("expected compile error for assignment to named vararg, got nil")
	}
	if !strings.Contains(compileErr.Error(), "const variable 't'") {
		t.Fatalf("expected 'const variable' error, got: %v", compileErr)
	}
}

func TestNamedVarArgConstUpvalue(t *testing.T) {
	// Named vararg captured as upvalue should also be const
	code := `
		local function foo(... extra)
			return function(...) extra = nil end
		end
	`
	block, err := parser.Parse("test", code)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	_, compileErr := compiler.Compile("test", block)
	if compileErr == nil {
		t.Fatalf("expected compile error for assignment to captured named vararg, got nil")
	}
	if !strings.Contains(compileErr.Error(), "const variable 'extra'") {
		t.Fatalf("expected 'const variable' error, got: %v", compileErr)
	}
}

func TestNamedVarArgNestedFunctions(t *testing.T) {
	_, lines, err := runLua(t, `
		local function foo(... tab1)
			return function(... tab2)
				return tab1, tab2
			end
		end
		local inner = foo(10, 20, 30)
		local t1, t2 = inner("a", "b")
		print(t1.n, t1[1], t2.n, t2[1])
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 1 || lines[0] != "3\t10\t2\ta" {
		t.Fatalf("expected '3\\t10\\t2\\ta', got %v", lines)
	}
}

func TestNamedVarArgMethod(t *testing.T) {
	_, lines, err := runLua(t, `
		local t = {}
		function t:method(... args)
			print(type(args), args.n)
		end
		t:method(1, 2)
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 1 || lines[0] != "table\t2" {
		t.Fatalf("expected 'table\\t2', got %v", lines)
	}
}

func TestNamedVarArgMissingParams(t *testing.T) {
	_, lines, err := runLua(t, `
		function f(a, b, ... rest)
			print(a, b, rest.n)
		end
		f(1)
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 1 || lines[0] != "1\tnil\t0" {
		t.Fatalf("expected '1\\tnil\\t0', got %v", lines)
	}
}

func TestNamedVarArgAnonymousFunc(t *testing.T) {
	_, lines, err := runLua(t, `
		local f = function(... args)
			return args.n, args[1]
		end
		print(f(10, 20))
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 1 || lines[0] != "2\t10" {
		t.Fatalf("expected '2\\t10', got %v", lines)
	}
}

func TestNamedVarArgViaLoad(t *testing.T) {
	// Test that load() can compile and run named vararg functions
	_, lines, err := runLua(t, `
		local f = load("return function(... args) return args.n, args[1] end")()
		print(f("hello", "world"))
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 1 || lines[0] != "2\thello" {
		t.Fatalf("expected '2\\thello', got %v", lines)
	}
}

func TestNamedVarArgMutation(t *testing.T) {
	// Modifying the named vararg table should be visible via ...
	_, lines, err := runLua(t, `
		function f(... args)
			args[2] = 99
			print(select(2, ...))
		end
		f(1, 2, 3)
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 1 || lines[0] != "99\t3" {
		t.Fatalf("expected '99\\t3', got %v", lines)
	}
}

func TestNamedVarArgTableWithPairsOverwrite(t *testing.T) {
	// From Lua 5.5 test suite: overwrite vararg table entries via pairs
	_, lines, err := runLua(t, `
		local function aux(a, v, ... t)
			for k, val in pairs(v) do t[k] = val end
			return ...
		end
		local r1, r2, r3 = aux(10, {11, [3] = 33}, 1, 2, 3)
		print(r1, r2, r3)
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 1 || lines[0] != "11\t2\t33" {
		t.Fatalf("expected '11\\t2\\t33', got %v", lines)
	}
}

func TestNamedVarArgInvalidN(t *testing.T) {
	// Setting n to a negative value should error when ... is expanded
	_, _, err := runLua(t, `
		function f(... args)
			args.n = -1
			return ...
		end
		f(1, 2)
	`)
	if err == nil {
		t.Fatalf("expected error for negative n")
	}
	if !strings.Contains(err.Error(), "no proper 'n'") {
		t.Fatalf("expected 'no proper n' error, got: %v", err)
	}
}

func TestNamedVarArgFloatN(t *testing.T) {
	// Setting n to a float should error when ... is expanded
	_, _, err := runLua(t, `
		function f(... args)
			args.n = 1.0
			return ...
		end
		f(1, 2)
	`)
	if err == nil {
		t.Fatalf("expected error for float n")
	}
	if !strings.Contains(err.Error(), "no proper 'n'") {
		t.Fatalf("expected 'no proper n' error, got: %v", err)
	}
}

func TestNamedVarArgWriteToTable(t *testing.T) {
	// From Lua 5.5 test suite: writing to the vararg table
	_, lines, err := runLua(t, `
		local function foo(... t)
			t[1] = t[1] + 10
			return t[1]
		end
		print(foo(10, 30))
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 1 || lines[0] != "20" {
		t.Fatalf("expected '20', got %v", lines)
	}
}
