package compiler_test

// Regression tests for code-generation defects that produced wrong values
// rather than errors: a method name overwriting the receiver, unchecked
// register operands in multiple assignment, a block-exit jump skipping
// OP_CLOSE, parentheses turning a live local read into a snapshot, and a unary
// operator evaluating its operand into the destination local.
//
// Every expectation below was produced by /usr/bin/lua5.4.8 on identical
// source; each case was also confirmed to reproduce on this branch before the
// fix (e.g. `obj:zzz()` handed the callee the method-name string as self, and
// the 100-target assignment silently stored into unrelated registers).
//
// The tests live in an external test package so they can run the compiled
// chunks: package compiler cannot import vm (vm imports compiler).

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

// runChunk compiles and runs src, returning its first return value as a string.
func runChunk(t *testing.T, src string) string {
	t.Helper()
	block, err := parser.Parse("codegen", src)
	if err != nil {
		t.Fatalf("parse error: %v\nsource:\n%s", err, src)
	}
	proto, err := compiler.Compile("codegen", block)
	if err != nil {
		t.Fatalf("compile error: %v\nsource:\n%s", err, src)
	}
	v := vm.New()
	stdlib.Open(v)
	res, err := v.Run(proto)
	if err != nil {
		t.Fatalf("runtime error: %v\nsource:\n%s", err, src)
	}
	if len(res) == 0 {
		return ""
	}
	return res[0].String()
}

// compileErr compiles src and returns the compile error message ("" on success).
func compileErr(t *testing.T, src string) string {
	t.Helper()
	block, err := parser.Parse("codegen", src)
	if err != nil {
		return err.Error()
	}
	if _, err := compiler.Compile("codegen", block); err != nil {
		return err.Error()
	}
	return ""
}

func checkChunk(t *testing.T, name, src, want string) {
	t.Helper()
	if got := runChunk(t, src); got != want {
		t.Errorf("%s: got %q, want %q", name, got, want)
	}
}

// TestSelfKeyBeyondArgC covers a method call whose name constant does not fit
// the SELF C operand: the fallback sequence must not allocate the method-name
// register on top of the receiver copy at base+1. Before the fix golua printed
// "OBJ string" here, i.e. the callee saw the method name as self.
func TestSelfKeyBeyondArgC(t *testing.T) {
	var b strings.Builder
	b.WriteString("local t = {}\n")
	for i := 0; i < 269; i++ {
		fmt.Fprintf(&b, "t.k%d = %q\n", i, fmt.Sprintf("v%d", i))
	}
	b.WriteString(`
local obj = setmetatable({}, {__index = {zzz = function(self) return "OBJ", type(self) end}})
local a, bb = obj:zzz()
return a .. " " .. bb
`)
	checkChunk(t, "self-key-beyond-argC", b.String(), "OBJ table")
}

// TestMultiAssignRegisterLimit covers multiple assignment overflowing the
// register file: the A/C operands are not masked, so an unchecked overflow
// silently stored into unrelated registers instead of raising. Reference
// lua5.4.8 rejects both programs with "function or expression needs too many
// registers"; golua used to run them and print "11 6" / "7 nil".
func TestMultiAssignRegisterLimit(t *testing.T) {
	build := func(rhs string) string {
		var b strings.Builder
		for i := 1; i <= 160; i++ {
			fmt.Fprintf(&b, "local v%d = %d\n", i, i)
		}
		b.WriteString("local T = {}\n")
		b.WriteString("local function f() return 11, 22, 33 end\n")
		for i := 1; i <= 100; i++ {
			if i > 1 {
				b.WriteString(",")
			}
			fmt.Fprintf(&b, "T[%d]", i)
		}
		b.WriteString(" = " + rhs + "\n")
		b.WriteString("return tostring(T[1]) .. ' ' .. tostring(T[100])\n")
		return b.String()
	}
	for _, rhs := range []string{"f(11, 22, 33)", "7"} {
		err := compileErr(t, build(rhs))
		if !strings.Contains(err, "too many registers") {
			t.Errorf("rhs %q: got error %q, want a \"too many registers\" compile error", rhs, err)
		}
	}
}

// TestBlockExitJumpClosesUpvalues covers a jump out of a block whose capture is
// only reached because a backward jump re-runs part of the block: the close
// must be decided when the block ends, not when the jump is compiled. Before
// the fix each of these returned "1 1 2" / "3 3 4" — the escaping closure read
// a stack slot that the locals after the block had reused.
func TestBlockExitJumpClosesUpvalues(t *testing.T) {
	checkChunk(t, "forward goto", `
local esc; local n = 0
do
  local x = "captured"
  ::again::
  n = n + 1
  if n > 1 then goto out end
  esc = function() return x end
  goto again
end
::out::
local p, q = 1, 2
return esc() .. " " .. p .. " " .. q
`, "captured 1 2")

	checkChunk(t, "backward goto", `
local esc, n = nil, 0
::top::
do
  local x = "cap"..n
  ::inner::
  n = n + 1
  if n == 3 then goto done end
  if n == 2 then goto top end
  esc = function() return x end
  goto inner
end
::done::
local p, q = 1, 2
return esc() .. " " .. p .. " " .. q
`, "cap0 1 2")

	checkChunk(t, "break", `
local esc, n = nil, 0
while true do
  do
    local x = "cap"
    ::again::
    n = n + 1
    if n > 1 then break end
    esc = function() return x end
    goto again
  end
end
local p, q = 1, 2
return esc() .. " " .. p .. " " .. q
`, "cap 1 2")

	checkChunk(t, "repeat break", `
local esc, m = nil, 0
repeat
  local y = "rep"
  ::more::
  m = m + 1
  if m > 1 then break end
  esc2 = nil
  esc = function() return y end
  goto more
until true
local r, s = 3, 4
return esc() .. " " .. r .. " " .. s
`, "rep 3 4")
}

// TestParenthesesAreInert covers '(' expr ')' around a local: parentheses only
// truncate multiple results, so the local must still be read live by the
// instruction that uses it rather than snapshotted into a temporary. The
// assignment cases stored into the WRONG TABLE before the fix.
func TestParenthesesAreInert(t *testing.T) {
	checkChunk(t, "arith operand", `
local a = 1
local function f() a = 100 return 5 end
return tostring((a) + f())
`, "105")

	checkChunk(t, "comparison operand", `
local b = 1
local function g() b = 100 return 5 end
return tostring((b) < g())
`, "false")

	checkChunk(t, "assignment table", `
local t, u = {}, {}
local x = t
local function h() x = u return "V" end
;(x).k = h()
return tostring(t.k) .. " " .. tostring(u.k)
`, "nil V")

	checkChunk(t, "assignment index", `
local t, u = {}, {}
local y = t
local function h() y = u return 7 end
;(y)[1] = h()
return tostring(t[1]) .. " " .. tostring(u[1])
`, "nil 7")

	checkChunk(t, "assignment key", `
local k1, k2 = 1, 2
local kk = k1
local tt = {}
local function h() kk = k2 return "W" end
tt[(kk)] = h()
return tostring(tt[1]) .. " " .. tostring(tt[2])
`, "nil W")

	checkChunk(t, "loop condition", `
local a, n = 0, 0
local function f() a = a + 1 return 2 end
while (a) < f() do n = n + 1 if n > 9 then break end end
return tostring(n)
`, "1")

	// Parentheses must still truncate multiple results to exactly one.
	checkChunk(t, "truncation", `
local function two() return "p", "q" end
local function count(...) return select('#', ...) end
return tostring(count((two()))) .. " " .. tostring(count(two()))
`, "1 2")
}

// TestUnopOperandNotInTarget covers unary -, # and ~ storing into a local: the
// operand must be evaluated in a temporary, so the target still holds its old
// value while a metamethod runs (and error messages blame the operand). Before
// the fix the metamethods observed the operand table in the target local.
func TestUnopOperandNotInTarget(t *testing.T) {
	checkChunk(t, "__unm", `
local v = 1
local t = {setmetatable({}, {__unm = function() return "saw v = "..tostring(v) end})}
v = -t[1]
return v
`, "saw v = 1")

	checkChunk(t, "__len", `
local z = 3
local t = {setmetatable({}, {__len = function() return "saw z = "..tostring(z) end})}
z = #t[1]
return z
`, "saw z = 3")

	checkChunk(t, "__bnot", `
local q = 4
local t = {setmetatable({}, {__bnot = function() return "saw q = "..tostring(q) end})}
q = ~t[1]
return q
`, "saw q = 4")

	checkChunk(t, "error names the operand", `
local w = 2
local ok, err = pcall(function() w = -UNDEFINED_GLOBAL end)
return tostring(ok) .. " " .. tostring(err):match("%(.*%)")
`, "false (global 'UNDEFINED_GLOBAL')")
}

// ---------------------------------------------------------------------------
// Randomized differential generator
// ---------------------------------------------------------------------------

// genProgram builds a random program mixing parenthesised local reads, unary
// operators, method calls, multiple assignment and block-exit jumps. It appends
// every observation to R; the caller decides whether the chunk returns or
// prints the joined result, so the same source can be run in-process and by the
// reference interpreter.
func genProgram(rnd *rand.Rand) string {
	var b strings.Builder
	w := func(format string, args ...interface{}) {
		fmt.Fprintf(&b, format+"\n", args...)
	}
	w("local R = {}")
	w("local function put(...) local n = select('#', ...) local p = {}")
	w("  for i = 1, n do p[i] = tostring((select(i, ...))) end")
	w("  R[#R+1] = table.concat(p, ',') end")
	w("local T = {1, 2, 3}")
	nv := 2 + rnd.Intn(3)
	for i := 0; i < nv; i++ {
		w("local v%d = %d", i, 1+rnd.Intn(9))
	}
	for i := 0; i < nv; i++ {
		w("local function m%d(r) v%d = v%d + 10 return r end", i, i, i)
	}
	local := func() string { return fmt.Sprintf("v%d", rnd.Intn(nv)) }
	operand := func() string {
		switch rnd.Intn(5) {
		case 0:
			return "(" + local() + ")"
		case 1:
			return "((" + local() + "))"
		case 2:
			return local()
		case 3:
			return fmt.Sprintf("m%d(%d)", rnd.Intn(nv), 1+rnd.Intn(5))
		default:
			return fmt.Sprintf("%d", 1+rnd.Intn(9))
		}
	}
	for i, n := 0, 3+rnd.Intn(6); i < n; i++ {
		switch rnd.Intn(8) {
		case 0:
			ops := []string{"+", "-", "*", "//", "%"}
			w("put(%s %s %s)", operand(), ops[rnd.Intn(len(ops))], operand())
		case 1:
			ops := []string{"<", "<=", ">", ">=", "==", "~="}
			w("put(%s %s %s)", operand(), ops[rnd.Intn(len(ops))], operand())
		case 2:
			ops := []string{"-", "#T +", "~", "not"}
			op := ops[rnd.Intn(len(ops))]
			if op == "#T +" {
				w("put(#T + (%s))", operand())
			} else {
				w("put(%s (%s))", op, operand())
			}
		case 3:
			mm := []string{"__unm", "__len", "__bnot"}[rnd.Intn(3)]
			sym := map[string]string{"__unm": "-", "__len": "#", "__bnot": "~"}[mm]
			n := local()
			w("do local mt = setmetatable({}, {%s = function() return 'saw '..tostring(%s) end})", mm, n)
			w("  local box = {mt}")
			w("  %s = %sbox[1]", n, sym)
			w("  put(%s)", n)
			w("  %s = %d", n, 1+rnd.Intn(9))
			w("end")
		case 4:
			w("do local o = {n = %d} function o:mm(x) return self.n, x end", 1+rnd.Intn(9))
			recv := []string{"o", "(o)"}[rnd.Intn(2)]
			w("  put(%s:mm(%s))", recv, operand())
			w("end")
		case 5:
			w("do local t1, t2 = {}, {}")
			w("  local sel = t1")
			w("  local function sw(r) sel = t2 return r end")
			tgt := []string{"(sel).k", "(sel)[1]", "sel.k"}[rnd.Intn(3)]
			w("  %s, t1.z = sw(%d), %d", tgt, 1+rnd.Intn(9), 1+rnd.Intn(9))
			w("  put(t1.k, t1[1], t2.k, t2[1], t1.z)")
			w("end")
		case 6:
			w("do local esc, cnt = nil, 0")
			w("  do")
			w("    local cx = 'cap%d'", i)
			w("    ::again%d::", i)
			w("    cnt = cnt + 1")
			w("    if cnt > 1 then goto out%d end", i)
			w("    esc = function() return cx end")
			w("    goto again%d", i)
			w("  end")
			w("  ::out%d::", i)
			w("  local f1, f2 = 1, 2")
			w("  put(esc(), f1, f2)")
			w("end")
		default:
			w("do local esc, cnt = nil, 0")
			w("  while true do")
			w("    do")
			w("      local bx = 'brk%d'", i)
			w("      ::more%d::", i)
			w("      cnt = cnt + 1")
			w("      if cnt > 1 then break end")
			w("      esc = function() return bx end")
			w("      goto more%d", i)
			w("    end")
			w("  end")
			w("  local g1, g2 = 3, 4")
			w("  put(esc(), g1, g2)")
			w("end")
		}
	}
	w("local out = table.concat(R, '\\n')")
	return b.String()
}

// referenceLua locates the PUC-Rio Lua 5.4.8 binary this branch is measured
// against. /usr/bin/lua is deliberately not accepted: it is 5.4.6 on the
// development host.
func referenceLua() string {
	for _, name := range []string{"lua5.4.8"} {
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		out, err := exec.Command(path, "-v").CombinedOutput()
		if err == nil && strings.Contains(string(out), "Lua 5.4.8") {
			return path
		}
	}
	return ""
}

// TestCodegenRandomDifferential generates programs from the shapes that this
// bug family keeps reappearing in and compares golua against the reference
// interpreter. It is skipped when no Lua 5.4.8 binary is installed.
func TestCodegenRandomDifferential(t *testing.T) {
	ref := referenceLua()
	if ref == "" {
		t.Skip("no PUC-Rio Lua 5.4.8 binary found; skipping differential run")
	}
	dir := t.TempDir()
	rnd := rand.New(rand.NewSource(20260810))
	for i := 0; i < 120; i++ {
		body := genProgram(rnd)
		got := runChunk(t, body+"\nreturn out\n")

		path := filepath.Join(dir, fmt.Sprintf("case%d.lua", i))
		if err := os.WriteFile(path, []byte(body+"\nio.write(out)\n"), 0o600); err != nil {
			t.Fatalf("write case: %v", err)
		}
		out, err := exec.Command(ref, path).CombinedOutput()
		if err != nil {
			t.Fatalf("case %d: reference failed: %v\n%s\nsource:\n%s", i, err, out, body)
		}
		if want := string(out); got != want {
			t.Fatalf("case %d: golua/reference mismatch\ngolua:\n%s\nreference:\n%s\nsource:\n%s",
				i, got, want, body)
		}
	}
}
