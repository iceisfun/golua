package compiler_test

// Format compatibility with PUC-Rio Lua 5.5 binary chunks.
//
// GoLua's master branch targets Lua 5.5, so its binary chunks must be Lua 5.5
// chunks: the layout of ldump.c/lundump.c, not merely the 5.5 header. These
// tests pin both directions — luac5.5.0 output must load and run here, and a
// chunk that came from luac must dump back byte for byte.
//
// The format carries nothing GoLua-specific: what used to need a private
// vararg bitfield (a named vararg, "... name") is a Lua 5.5 construct with a
// flag of its own, PF_VATAB.
//
// Reference bytecode is never executed by reference Lua here, and GoLua's
// dumps are never handed to it: reference ships no bytecode verifier, so
// feeding it foreign chunks is a segfault waiting to happen. Only bytes on
// disk are compared.

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

const luacBin = "luac5.5.0"

// requireLuac skips the test when the reference compiler is not installed.
func requireLuac(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath(luacBin)
	if err != nil {
		t.Skipf("%s not available: %v", luacBin, err)
	}
	return path
}

// compileChunk compiles src as a main chunk named chunkName.
func compileChunk(t *testing.T, chunkName, src string) *compiler.Proto {
	t.Helper()
	block, err := parser.Parse(chunkName, src)
	if err != nil {
		t.Fatalf("parse %s: %v", chunkName, err)
	}
	proto, err := compiler.Compile(chunkName, block)
	if err != nil {
		t.Fatalf("compile %s: %v", chunkName, err)
	}
	return proto
}

// runLua runs src in a fresh VM with the standard library open, passing args as
// the chunk's varargs. It returns the chunk's results and everything print
// wrote.
func runLua(t *testing.T, src string, args ...vm.Value) ([]vm.Value, string) {
	t.Helper()
	v := vm.New(vm.WithCaptureOutput(true))
	v.SetDebugProvider(vm.NewDefaultDebugProvider())
	stdlib.Open(v)
	res, err := v.RunArgs(compileChunk(t, "=test", src), args)
	if err != nil {
		t.Fatalf("run: %v\nsource:\n%s", err, src)
	}
	return res, strings.Join(v.OutputLines(), "\n")
}

// luaOutput runs src and returns only what it printed.
func luaOutput(t *testing.T, src string, args ...vm.Value) string {
	t.Helper()
	_, out := runLua(t, src, args...)
	return out
}

// luaString runs src and returns its single string result.
func luaString(t *testing.T, src string, args ...vm.Value) string {
	t.Helper()
	res, _ := runLua(t, src, args...)
	if len(res) != 1 || !res[0].IsString() {
		t.Fatalf("expected one string result, got %v", res)
	}
	return res[0].AsString()
}

// luacDump compiles src with luac5.5.0 and returns the binary chunk it wrote.
// The source file lives in the test's temp dir; its path becomes the chunk's
// source name ("@" + path), which the GoLua side reproduces exactly.
func luacDump(t *testing.T, src string, strip bool) (chunk []byte, srcPath string) {
	t.Helper()
	luac := requireLuac(t)
	dir := t.TempDir()
	srcPath = filepath.Join(dir, "chunk.lua")
	if err := os.WriteFile(srcPath, []byte(src), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(dir, "chunk.out")
	args := []string{"-o", outPath}
	if strip {
		args = append(args, "-s")
	}
	args = append(args, srcPath)
	cmd := exec.Command(luac, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s %v: %v\n%s", luac, args, err, stderr.String())
	}
	chunk, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read %s output: %v", luacBin, err)
	}
	return chunk, srcPath
}

// goluaDump compiles src as a main chunk named chunkName and returns
// string.dump of it: GoLua's own binary chunk for the same program.
func goluaDump(t *testing.T, chunkName, src string, strip bool) []byte {
	t.Helper()
	proto := compileChunk(t, chunkName, src)
	cl := vm.NewClosure(proto)
	for i := range proto.Upvalues {
		cl.Upvalues[i] = &vm.Upvalue{}
	}
	// string.dump is only reachable from Lua, so call it from a driver chunk
	// that receives the target function as a vararg.
	dump := luaString(t, "local f, strip = ...\nreturn string.dump(f, strip)\n",
		vm.NewFunction(cl), vm.NewBool(strip))
	return []byte(dump)
}

// reDump loads a binary chunk and dumps the loaded prototype again.
func reDump(t *testing.T, chunk []byte, strip bool) []byte {
	t.Helper()
	dump := luaString(t, "local data, strip = ...\nreturn string.dump(assert(load(data)), strip)\n",
		vm.NewString(string(chunk)), vm.NewBool(strip))
	return []byte(dump)
}

// firstDiff describes where two chunks diverge, with a little context.
func firstDiff(got, want []byte) string {
	n := min(len(got), len(want))
	for i := 0; i < n; i++ {
		if got[i] != want[i] {
			lo := max(i-8, 0)
			return fmt.Sprintf("first difference at byte %d (0x%x): got 0x%02x, want 0x%02x\n got  % x\n want % x",
				i, i, got[i], want[i], got[lo:min(i+8, len(got))], want[lo:min(i+8, len(want))])
		}
	}
	return fmt.Sprintf("common prefix identical; lengths differ: got %d, want %d", len(got), len(want))
}

// compatCorpus holds chunks that use no GoLua-specific extension, so their
// dumps must match luac5.5.0's byte for byte.
var compatCorpus = []struct {
	name string
	src  string
}{
	{"empty", ""},
	{"return_constant", "return 42\n"},
	{"plain_function", "local function f(x) return x + 1 end\nreturn f(1)\n"},
	{"upvalues", "local a = 1\nlocal function f(x) a = a + x return a end\nreturn f(2)\n"},
	{"nested_closures", `
local function outer(n)
  local function middle(m)
    return function(k) return n + m + k end
  end
  return middle
end
return outer(1)(2)(3)
`},
	{"all_constant_types", `
local function f()
  return nil, true, false, 42, -100, 0, 3.14, -0.5, 1e300, "hi",
         "a string that is definitely longer than forty characters long"
end
return f()
`},
	{"integer_constants", `
local t = {2^53, -9007199254740993, 0x7fffffffffffffff, -1, 127, -128, 300}
return t
`},
	{"table_and_calls", `
local t = {1, 2, 3, x = "y", [10] = true}
local s = 0
for i, v in ipairs(t) do s = s + i end
for k, v in pairs(t) do s = s + 1 end
return s, #t, t.x
`},
	{"control_flow", `
local s = 0
for i = 1, 10 do
  if i % 2 == 0 then s = s + i elseif i == 5 then s = s - 1 else s = s + 1 end
end
local i = 0
while i < 10 do i = i + 1 if i == 7 then break end end
repeat i = i - 1 until i == 0
do goto done end
::done::
return s
`},
	{"metamethod_ops", `
local function f(a, b)
  return a + b, a - b, a * b, a / b, a // b, a % b, a ^ b,
         a & b, a | b, a ~ b, a << b, a >> b, -a, ~a, #a, a .. b
end
return f(3, 4)
`},
	{"comparisons", `
local function f(a, b)
  if a < b then return 1 elseif a <= 2 then return 2 elseif a > 3 then return 3
  elseif a >= b then return 4 elseif a == b then return 5 end
  return 0
end
return f(1, 2)
`},
	{"varargs", `
local function f(...)
  local a, b = ...
  return select("#", ...), a, b, {...}
end
return f(1, 2, 3)
`},
	{"method_and_self", `
local obj = {}
function obj:m(x) return self, x end
function obj.n(x) return x end
return obj:m(1), obj.n(2)
`},
	{"tbc_and_const", `
local x <const> = 5
do
  local y <close> = setmetatable({}, {__close = function() end})
end
return x
`},
	{"upvalue_kind_const", `
-- An upvalue captured from a "<const>" local whose value is not a
-- compile-time constant: reference records the upvalue's kind as RDKCONST.
local x <const> = {1, 2}
local function f() return x[1] end
return f()
`},
	{"upvalue_kind_tbc", `
-- An upvalue captured from a to-be-closed local: kind RDKTOCLOSE.
local r
do
  local y <close> = setmetatable({}, {__close = function() end})
  local function g() return y end
  r = g()
end
return r
`},
	{"upvalue_kind_vararg_param", `
-- An upvalue captured from a named vararg parameter: kind RDKVAVAR. Capturing
-- it is also what makes reference materialize the vararg table (lcode.c
-- needvatab, reached through singlevaraux -> luaK_vapar2local), so the
-- enclosing function is PF_VATAB.
local function f(a, ... args)
  return function() return args[1], args.n end
end
return f(1, 2, 3)()
`},
	{"upvalue_kinds_mixed", `
-- All the kinds at once, including one captured through two levels: newupvalue
-- copies the kind from the enclosing function's upvalue, so it must survive
-- more than one hop.
local c <const> = {}
local function outer()
  local function inner() return c end
  return inner
end
return outer()
`},
	{"named_vararg_table", `
-- "... name" is a Lua 5.5 feature, not a GoLua extension: reference collects
-- the extra arguments into a vararg table in the register after the fixed
-- parameters and flags the prototype PF_VATAB, which is exactly what GoLua
-- does. Binding it to a local forces reference to materialize that table
-- rather than keeping the arguments hidden below the frame.
local function f(a, ... args)
  local t = args
  return a, t.n, t[1], t[2], select("#", ...)
end
return f(1, 2, 3)
`},
	{"shift_immediates", `
-- OP_SHLI/OP_SHRI: GoLua's own code generator never emits these, so they
-- exercise the opcode translation on a chunk that came from luac.
local function f(x)
  return x << 4, x >> 4, 1 << x, 256 >> x, x << 63, x >> -2
end
return f(0xFF00)
`},
	{"big_table_constructor", "local t = {" + strings.Repeat("1,", 2000) + "}\nreturn #t\n"},
	{"big_line_gaps", "local a = 1\n" + strings.Repeat("\n", 400) + "local b = 2\n" +
		strings.Repeat("\n", 400) + "return a + b\n"},
	{"many_instructions", "local s = 0\n" + strings.Repeat("s = s + 1\n", 300) + "return s\n"},
	{"long_strings", `
local a = [[` + strings.Repeat("x", 200) + `]]
local b = "` + strings.Repeat("y", 41) + `"
local c = "` + strings.Repeat("z", 40) + `"
return a, b, c
`},
}

// TestDumpOfLuacChunkIsByteIdentical is the primary compatibility pin. It
// takes a chunk produced by luac5.5.0, loads it, and dumps it again: the bytes
// must come back exactly as luac wrote them. That covers every part of the
// container in both directions — MSB-first varints, zigzag integers, the
// 4-byte alignment before the code and abslineinfo vectors, the string-reuse
// table and the order strings enter it, the flag byte, the field order (debug
// information and the source name after the nested prototypes), the delta line
// encoding with its ABSLINEINFO escapes, and the metamethod tags in
// OP_MMBIN*. Nothing but a genuine Lua 5.5 implementation of the format can
// reproduce a reference chunk byte for byte.
//
// GoLua's dump of the *same source* is a different question: it depends on the
// code generator, and GoLua's differs from luac's in ways this change does not
// touch (see TestDumpVsLuacDivergesOnlyInGeneratedCode).
func TestDumpOfLuacChunkIsByteIdentical(t *testing.T) {
	requireLuac(t)
	for _, tc := range compatCorpus {
		for _, strip := range []bool{false, true} {
			name := tc.name
			if strip {
				name += "_stripped"
			}
			t.Run(name, func(t *testing.T) {
				ref, _ := luacDump(t, tc.src, strip)
				got := reDump(t, ref, strip)
				if !bytes.Equal(got, ref) {
					t.Errorf("re-dump differs from %s (golua %d bytes, luac %d bytes)\n%s",
						luacBin, len(got), len(ref), firstDiff(got, ref))
				}
			})
		}
	}
}

// TestDumpVsLuacDivergesOnlyInGeneratedCode records the state of the other
// direction: GoLua's dump of a source file is not byte-identical to luac's,
// because the two code generators emit different instructions — GoLua has no
// equivalent of luaK_finish's rewrite of a vararg function's final
// RETURN0/RETURN1 into RETURN with C = numparams+1 (its VM keeps varargs in a
// Go slice instead of hidden stack slots), it does not append the extra RETURN
// after a tail call, and it assigns OP_VARARGPREP a different line. Those are
// code-generator differences, not format differences: everything the container
// carries about a function matches field for field.
//
// This test pins that boundary, so a future divergence in the *format* cannot
// hide behind the known code-generator gap.
func TestDumpVsLuacDivergesOnlyInGeneratedCode(t *testing.T) {
	requireLuac(t)
	for _, tc := range compatCorpus {
		t.Run(tc.name, func(t *testing.T) {
			ref, srcPath := luacDump(t, tc.src, false)
			got := goluaDump(t, "@"+srcPath, tc.src, false)
			refProto, refUp, err := compiler.Undump(ref, "ref")
			if err != nil {
				t.Fatalf("undump %s chunk: %v", luacBin, err)
			}
			gotProto, gotUp, err := compiler.Undump(got, "got")
			if err != nil {
				t.Fatalf("undump golua chunk: %v", err)
			}
			if refUp != gotUp {
				t.Errorf("top-level upvalue count: golua %d, %s %d", gotUp, luacBin, refUp)
			}
			compareProtoMetadata(t, "main", gotProto, refProto)
		})
	}
}

// compareProtoMetadata checks every prototype field the binary format carries
// that does not depend on the instructions chosen by the code generator.
func compareProtoMetadata(t *testing.T, path string, got, want *compiler.Proto) {
	t.Helper()
	if got.Source != want.Source {
		t.Errorf("%s: source %q != %q", path, got.Source, want.Source)
	}
	if got.LineDef != want.LineDef || got.LastLine != want.LastLine {
		t.Errorf("%s: lines %d,%d != %d,%d", path, got.LineDef, got.LastLine, want.LineDef, want.LastLine)
	}
	if got.NumParams != want.NumParams {
		t.Errorf("%s: numparams %d != %d", path, got.NumParams, want.NumParams)
	}
	if got.IsVarArg != want.IsVarArg || got.HasNamedVarArg != want.HasNamedVarArg {
		t.Errorf("%s: vararg flags (%v,%v) != (%v,%v)", path,
			got.IsVarArg, got.HasNamedVarArg, want.IsVarArg, want.HasNamedVarArg)
	}
	if len(got.Upvalues) != len(want.Upvalues) {
		t.Errorf("%s: %d upvalues != %d", path, len(got.Upvalues), len(want.Upvalues))
	} else {
		for i := range got.Upvalues {
			g, w := got.Upvalues[i], want.Upvalues[i]
			if g.Name != w.Name || g.InStack != w.InStack || g.Index != w.Index {
				t.Errorf("%s: upvalue %d %+v != %+v", path, i, g, w)
			}
		}
	}
	// The constant pool is not compared: which values reach it, and in which
	// order, is a code-generator decision (GoLua folds a negated literal into
	// the constant, orders table-constructor keys differently, and so on).
	if len(got.Protos) != len(want.Protos) {
		t.Fatalf("%s: %d nested prototypes != %d", path, len(got.Protos), len(want.Protos))
	}
	for i := range got.Protos {
		compareProtoMetadata(t, fmt.Sprintf("%s[%d]", path, i), got.Protos[i], want.Protos[i])
	}
}

// TestLoadLuacChunk checks that chunks precompiled by luac5.5.0 load and run
// with the right results, stripped and unstripped.
func TestLoadLuacChunk(t *testing.T) {
	requireLuac(t)
	cases := []struct{ name, src, want string }{
		{"arith", `print(1 + 2, 7 // 2, 2^10, "a" .. "b")`, "3\t3\t1024.0\tab"},
		{"closures", `
local function counter()
  local n = 0
  return function() n = n + 1 return n end
end
local c = counter()
c() c()
print(c())
`, "3"},
		{"varargs_and_table", `
local function f(...) return select("#", ...), ... end
local n, a, b = f(10, 20)
local t = {}
for i = 1, 5 do t[i] = i * i end
print(n, a, b, #t, t[5])
`, "2\t10\t20\t5\t25"},
		{"strings_and_metatables", `
local mt = {__index = function(_, k) return k .. "!" end}
local t = setmetatable({}, mt)
print(t.hello, ("%d/%s"):format(7, "x"), string.rep("ab", 3, "-"))
`, "hello!\t7/x\tab-ab-ab"},
		{"metamethod_dispatch", `
local mt = {__add = function() return "added" end,
            __lt = function() return true end,
            __concat = function() return "cat" end}
local a = setmetatable({}, mt)
print(a + 1, 1 + a, a < a, a .. "x")
`, "added\tadded\ttrue\tcat"},
		{"deep_recursion", `
local function fib(n) if n < 2 then return n end return fib(n-1) + fib(n-2) end
print(fib(20))
`, "6765"},
		{"long_constants", `
local s = string.rep("q", 100)
print(#s, s == string.rep("q", 100))
`, "100\ttrue"},
		{"shift_immediates", `
local x = 0xFF00
print(x >> 4, x << 4, 1 << 8, 256 >> 4, x >> 100, x << -4)
`, "4080\t1044480\t256\t16\t0\t4080"},
		{"big_table_constructor", `
local t = {` + strings.Repeat("1,", 2000) + `}
print(#t, t[1], t[1500], t[2000], t[2001])
`, "2000\t1\t1\t1\tnil"},
		{"named_vararg", `
-- A reference chunk whose vararg parameter is materialized as a table
-- (PF_VATAB) is the same representation GoLua gives "... name".
local function f(a, ... args)
  local t = args
  return a, t.n, t[1], select("#", ...)
end
print(f(1, 2, 3))
`, "1\t2\t2\t2"},
		{"generic_for", `
local t = {10, 20, 30, a = 1}
local s = 0
for i, v in ipairs(t) do s = s + v end
for k, v in pairs(t) do s = s + 1 end
local n = 0
for w in ("a b c"):gmatch("%a") do n = n + 1 end
print(s, n)
`, "64\t3"},
	}
	for _, tc := range cases {
		for _, strip := range []bool{false, true} {
			name := tc.name
			if strip {
				name += "_stripped"
			}
			t.Run(name, func(t *testing.T) {
				chunk, _ := luacDump(t, tc.src, strip)
				out := luaOutput(t, "local data = ...\nassert(load(data))()\n",
					vm.NewString(string(chunk)))
				if out != tc.want {
					t.Errorf("running %s chunk: got %q, want %q", luacBin, out, tc.want)
				}
			})
		}
	}
}

// TestLoadLuacChunkDebugInfo checks that the debug information of a reference
// chunk survives the load: line numbers, local names and upvalue names. A
// stripped chunk carries none of it, and must then report errors the way
// reference does — the bare message, with no position at all.
func TestLoadLuacChunkDebugInfo(t *testing.T) {
	requireLuac(t)
	src := `
local up = 1
local f
f = function(a, b)
  local c = a + b + up
  print(debug.getinfo(1, "Sl").currentline, debug.getlocal(1, 1), debug.getupvalue(f, 1))
end
f(1, 2)
`
	chunk, srcPath := luacDump(t, src, false)
	out := luaOutput(t, "local data = ...\nassert(load(data))()\n", vm.NewString(string(chunk)))
	if want := "6\ta\tup\t1"; out != want {
		t.Errorf("debug info from %s chunk: got %q, want %q", luacBin, out, want)
	}

	// Error locations must name the file the chunk was compiled from.
	errSrc := "local function boom() error('kaboom') end\nboom()\n"
	chunk, srcPath = luacDump(t, errSrc, false)
	out = luaOutput(t, "local data = ...\nprint(select(2, pcall(assert(load(data)))))\n",
		vm.NewString(string(chunk)))
	if want := srcPath + ":1: kaboom"; out != want {
		t.Errorf("error location: got %q, want %q", out, want)
	}

	// A stripped chunk has neither source nor line information, so reference
	// reports the bare message with no position; GoLua must do the same.
	chunk, _ = luacDump(t, errSrc, true)
	out = luaOutput(t, "local data = ...\nprint(select(2, pcall(assert(load(data)))))\n",
		vm.NewString(string(chunk)))
	if out != "kaboom" {
		t.Errorf("stripped error location: got %q, want %q", out, "kaboom")
	}
}

// TestNamedVarargRoundTrip covers the construct that used to cost the format a
// private bitfield bit plus an extra byte: a named vararg ("... name"). It is
// dumped as Lua 5.5's PF_VATAB — a vararg table in the register after the
// fixed parameters — which is both what reference means by that flag and what
// GoLua's VM does, so nothing GoLua-specific enters the layout and the
// register is implied by numparams.
//
// This covers the PF_VATAB half of the construct. Reference keeps a named
// vararg's arguments hidden below the frame (PF_VAHID) by default and reads
// them with OP_GETVARG; lcode.c needvatab promotes it to PF_VATAB only when the
// parameter is used as a plain value, which is what "local t = args" below does.
// The other half is TestNamedVarargBothForms.
func TestNamedVarargRoundTrip(t *testing.T) {
	for _, strip := range []string{"false", "true"} {
		t.Run("strip="+strip, func(t *testing.T) {
			out := luaOutput(t, `
local function f(a, ... args)
  return a, args.n, args[1], args[2], select("#", ...), (select(2, ...))
end
local g = assert(load(string.dump(f, `+strip+`)))
print(g(1, 2, 3))
print(f(1, 2, 3))
`)
			if want := "1\t2\t2\t3\t2\t3\n1\t2\t2\t3\t2\t3"; out != want {
				t.Errorf("named vararg round-trip: got %q, want %q", out, want)
			}
		})
	}
}

// TestVarargSlotRoundTrip pins the plain-"..." case: the register reference
// reserves for the (unused) vararg table must still read back as nil after a
// round-trip, exactly as for a freshly compiled function.
func TestVarargSlotRoundTrip(t *testing.T) {
	out := luaOutput(t, `
local function f(a, ...)
  local name, value = debug.getlocal(1, 2)
  return name, value, select("#", ...)
end
local g = assert(load(string.dump(f)))
print(f(1, 2, 3))
print(g(1, 2, 3))
`)
	if want := "(vararg table)\tnil\t2\n(vararg table)\tnil\t2"; out != want {
		t.Errorf("vararg slot round-trip: got %q, want %q", out, want)
	}
}

// TestSelfRoundTrip exercises constructs whose encodings are easy to get wrong
// (all constant types, deep proto nesting, abslineinfo, stripped dumps)
// through GoLua's own dump/load path: loading a dump and dumping it again must
// reproduce the same bytes, so no field was lost or reinterpreted.
func TestSelfRoundTrip(t *testing.T) {
	for _, tc := range compatCorpus {
		t.Run(tc.name, func(t *testing.T) {
			for _, strip := range []bool{false, true} {
				chunk := goluaDump(t, "@self.lua", tc.src, strip)
				proto, nUp, err := compiler.Undump(chunk, "@self.lua")
				if err != nil {
					t.Fatalf("undump (strip=%v): %v", strip, err)
				}
				if nUp != len(proto.Upvalues) {
					t.Fatalf("upvalue count %d != %d", nUp, len(proto.Upvalues))
				}
				if again := reDump(t, chunk, strip); !bytes.Equal(again, chunk) {
					t.Errorf("re-dump differs (strip=%v)\n%s", strip, firstDiff(again, chunk))
				}
			}
		})
	}
}

// TestRoundTripResults runs a battery of constructs through dump/load and
// checks that the loaded chunk still behaves.
func TestRoundTripResults(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"constants", `
local function f() return nil, true, false, 42, -7, 3.5, 2^53, "s",
  "` + strings.Repeat("L", 100) + `" end
local g = assert(load(string.dump(f)))
print(g())
`, "nil\ttrue\tfalse\t42\t-7\t3.5\t9007199254740992.0\ts\t" + strings.Repeat("L", 100)},
		{"upvalues", `
-- load() binds the first upvalue of a loaded chunk to _ENV, so a dumped
-- function reaches its globals; other upvalues come back nil, as in reference.
x = 10
local function f(n) x = x + n return x end
local g = assert(load(string.dump(f)))
print(f(1), g(1), x)
`, "11\t12\t12"},
		{"nested", `
local function f(a) return function(b) return function(c) return a + b + c end end end
local g = assert(load(string.dump(f)))
print(g(1)(2)(3))
`, "6"},
		{"upvalue_names", `
local up = 1
local function f() return up end
local g = assert(load(string.dump(f)))
local stripped = assert(load(string.dump(f, true)))
print((debug.getupvalue(g, 1)), (debug.getupvalue(stripped, 1)))
`, "up\t(no name)"},
		{"long_source_name", `
local name = "@" .. string.rep("d/", 200) .. "chunk.lua"
local f = assert(load("local function g() error('x') end return g", name))()
local h = assert(load(string.dump(f)))
local ok, err = pcall(h)
print(err:sub(-24))
`, "d/d/d/d/d/chunk.lua:1: x"},
		{"stripped_lines", `
-- Reference reports no position at all for a stripped function.
local function f() error("x") end
local g = assert(load(string.dump(f, true)))
local ok, err = pcall(g)
print(ok, err)
`, "false\tx"},
		{"abslineinfo", `
local src = "local a = 1\n" .. string.rep("\n", 500) .. "error('deep')\n"
local f = assert(load(src, "=big"))
local g = assert(load(string.dump(f)))
local ok, err = pcall(g)
print(err)
`, "big:502: deep"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if out := luaOutput(t, tc.src); out != tc.want {
				t.Errorf("got %q, want %q", out, tc.want)
			}
		})
	}
}

// TestHostileChunksAreCatchable pins the hardening: a truncated, corrupt or
// deliberately inflated chunk must surface as an ordinary "bad binary format"
// error, never a fatal allocation or a host crash. Executing untrusted
// bytecode remains unsafe by design (see wontfix/untrusted-binary-chunks);
// loading it must not be.
func TestHostileChunksAreCatchable(t *testing.T) {
	good := goluaDump(t, "@hostile.lua", `
local t = {1, 2, 3}
local function f(a, b) return a + b + #t end
return f(1, 2)
`, false)

	t.Run("truncated_at_every_length", func(t *testing.T) {
		for n := 0; n < len(good); n++ {
			proto, _, err := compiler.Undump(good[:n], "trunc")
			if err == nil {
				t.Fatalf("prefix of %d bytes loaded without error", n)
			}
			if proto != nil {
				t.Fatalf("prefix of %d bytes returned a proto alongside an error", n)
			}
			if !strings.Contains(err.Error(), "bad binary format") {
				t.Fatalf("prefix of %d bytes: unexpected error %v", n, err)
			}
		}
	})

	t.Run("single_byte_corruption", func(t *testing.T) {
		// Flipping any single byte must either load (a valid but different
		// chunk) or fail cleanly; it must never escape Undump as a panic or a
		// non-"bad binary format" error.
		for i := 0; i < len(good); i++ {
			for _, mask := range []byte{0xff, 0x80, 0x01} {
				corrupt := append([]byte(nil), good...)
				corrupt[i] ^= mask
				_, _, err := compiler.Undump(corrupt, "corrupt")
				if err != nil && !strings.Contains(err.Error(), "bad binary format") {
					t.Fatalf("byte %d ^ 0x%02x: unexpected error %v", i, mask, err)
				}
			}
		}
	})

	t.Run("inflated_counts", func(t *testing.T) {
		// A count that would drive a multi-gigabyte allocation must be
		// rejected against the bytes remaining, not attempted: an OOM here is
		// a Go fatal error that recover() cannot catch.
		const huge = 2_000_000_000
		for _, field := range []string{"code", "constants", "upvalues", "protos", "lineinfo", "abslineinfo", "locvars"} {
			data := chunkWithInflatedCount(t, field, huge)
			proto, _, err := compiler.Undump(data, "evil")
			if err == nil {
				t.Fatalf("%s: inflated count loaded (proto=%v)", field, proto)
			}
			if !strings.Contains(err.Error(), "bad binary format") {
				t.Fatalf("%s: unexpected error %v", field, err)
			}
		}
	})

	t.Run("huge_string_size", func(t *testing.T) {
		// A string whose declared size is enormous must be rejected before the
		// bytes are read, and a size near the integer limits must not wrap the
		// bounds check into a no-op.
		for _, size := range []uint64{1 << 40, 1<<63 - 1, 1 << 63, ^uint64(0)} {
			var buf bytes.Buffer
			buf.Write(validChunkHeader())
			putVarint(&buf, 0) // linedefined
			putVarint(&buf, 0) // lastlinedefined
			buf.WriteByte(0)   // numparams
			buf.WriteByte(0)   // flag
			buf.WriteByte(2)   // maxstacksize
			putVarint(&buf, 0) // sizecode
			for buf.Len()%4 != 0 {
				buf.WriteByte(0)
			}
			putVarint(&buf, 1)    // sizek
			buf.WriteByte(0x04)   // LUA_VSHRSTR
			putVarint(&buf, size) // string size
			if _, _, err := compiler.Undump(buf.Bytes(), "evil"); err == nil {
				t.Fatalf("string size %d loaded without error", size)
			}
		}
	})

	t.Run("garbage_after_header", func(t *testing.T) {
		for _, tail := range []string{"", "\x00", "garbage", strings.Repeat("\xff", 64)} {
			var buf bytes.Buffer
			buf.Write(validChunkHeader())
			buf.WriteString(tail)
			if _, _, err := compiler.Undump(buf.Bytes(), "evil"); err == nil {
				t.Fatalf("header + %q loaded without error", tail)
			}
		}
	})

	t.Run("load_from_lua_stays_catchable", func(t *testing.T) {
		// The same guarantee through the Lua surface: load() returns nil plus
		// a message instead of taking the host down.
		out := luaOutput(t, `
local data = ...
-- from 1: the empty string is a valid (text) chunk, not a truncated one.
for n = 1, #data - 1 do
  local f, err = load(data:sub(1, n))
  assert(f == nil, "truncated chunk loaded at " .. n)
  assert(type(err) == "string")
end
assert(load(data), "the whole chunk must still load")
print("ok")
`, vm.NewString(string(good)))
		if out != "ok" {
			t.Errorf("got %q", out)
		}
	})
}

// validChunkHeader builds the exact 5.5 header, through the top-level upvalue
// count, that the undumper accepts.
func validChunkHeader() []byte {
	var buf bytes.Buffer
	buf.WriteString("\x1bLua")
	buf.WriteByte(0x55) // version
	buf.WriteByte(0x00) // format
	buf.WriteString("\x19\x93\r\n\x1a\n")
	buf.WriteByte(4)
	buf.Write([]byte{0x88, 0xa9, 0xff, 0xff}) // (int) -0x5678
	buf.WriteByte(4)
	buf.Write([]byte{0x78, 0x56, 0x34, 0x12}) // (Instruction) 0x12345678
	buf.WriteByte(8)
	buf.Write([]byte{0x88, 0xa9, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}) // (lua_Integer) -0x5678
	buf.WriteByte(8)
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x28, 0x77, 0xc0}) // (lua_Number) -370.5
	buf.WriteByte(0x00)                                               // top-level upvalue count
	return buf.Bytes()
}

// putVarint appends x in Lua 5.5's MSB-first varint encoding (continuation bit
// on every byte but the last).
func putVarint(buf *bytes.Buffer, x uint64) {
	var tmp [10]byte
	n := 1
	tmp[9] = byte(x & 0x7f)
	for x >>= 7; x != 0; x >>= 7 {
		n++
		tmp[10-n] = byte(x&0x7f) | 0x80
	}
	buf.Write(tmp[10-n:])
}

// chunkWithInflatedCount builds a chunk whose named element count claims
// `count` entries while the chunk holds none of them.
func chunkWithInflatedCount(t *testing.T, field string, count uint64) []byte {
	t.Helper()
	var buf bytes.Buffer
	buf.Write(validChunkHeader())
	putVarint(&buf, 0) // linedefined
	putVarint(&buf, 0) // lastlinedefined
	buf.WriteByte(0)   // numparams
	buf.WriteByte(0)   // flag
	buf.WriteByte(2)   // maxstacksize

	if field == "code" {
		putVarint(&buf, count)
		return buf.Bytes()
	}
	putVarint(&buf, 0) // sizecode
	for buf.Len()%4 != 0 {
		buf.WriteByte(0) // alignment padding before the (empty) code vector
	}

	if field == "constants" {
		putVarint(&buf, count)
		return buf.Bytes()
	}
	putVarint(&buf, 0) // sizek

	if field == "upvalues" {
		putVarint(&buf, count)
		return buf.Bytes()
	}
	putVarint(&buf, 0) // sizeupvalues

	if field == "protos" {
		putVarint(&buf, count)
		return buf.Bytes()
	}
	putVarint(&buf, 0) // sizep
	putVarint(&buf, 0) // source: "no string" marker
	putVarint(&buf, 0)

	if field == "lineinfo" {
		putVarint(&buf, count)
		return buf.Bytes()
	}
	putVarint(&buf, 0) // sizelineinfo

	if field == "abslineinfo" {
		putVarint(&buf, count)
		return buf.Bytes()
	}
	putVarint(&buf, 0) // sizeabslineinfo

	if field == "locvars" {
		putVarint(&buf, count)
		return buf.Bytes()
	}
	t.Fatalf("unknown field %q", field)
	return nil
}

// ---------------------------------------------------------------------------
// Upvalue kinds
// ---------------------------------------------------------------------------

// Reference upvalue kinds (lparser.h). Only these four can reach an upvalue
// descriptor: RDKCTC variables are compile-time constants that never occupy a
// register, and the global kinds are not variables a closure can capture.
const (
	kindVDKREG     = 0
	kindRDKCONST   = 1
	kindRDKVAVAR   = 2
	kindRDKTOCLOSE = 3
)

// upvalueKinds collects the kind byte of every upvalue in a prototype tree, in
// dump order.
func upvalueKinds(p *compiler.Proto) []byte {
	var kinds []byte
	for _, uv := range p.Upvalues {
		kinds = append(kinds, uv.Kind)
	}
	for _, sub := range p.Protos {
		kinds = append(kinds, upvalueKinds(sub)...)
	}
	return kinds
}

// TestUpvalueKindIsPreserved pins a field the format carries and the loader
// used to drop. Each upvalue descriptor holds the kind of the variable that was
// captured, and luac emits a non-zero one whenever a closure captures a
// "<const>" local that is not a compile-time constant (RDKCONST), a named
// vararg parameter (RDKVAVAR) or a "<close>" local (RDKTOCLOSE). The undumper
// read the byte and threw it away while the dumper always wrote zero, so
// load-then-dump of such a chunk silently changed it and the byte-identical
// claim did not hold in general.
//
// Nothing at run time consults the kind — reference's lvm.c does not read it
// either — so the requirement is exactly that it survives untouched.
func TestUpvalueKindIsPreserved(t *testing.T) {
	requireLuac(t)
	cases := []struct {
		name string
		src  string
		want byte
	}{
		{"regular_local", `
local a = 1
local function f() return a end
return f()
`, kindVDKREG},
		{"const_local", `
local x <const> = {1}
local function f() return x[1] end
return f()
`, kindRDKCONST},
		{"vararg_parameter", `
local function f(a, ... args)
  return function() return args end
end
return f(1, 2)
`, kindRDKVAVAR},
		{"to_be_closed_local", `
local r
do
  local y <close> = setmetatable({}, {__close = function() end})
  local function g() return y end
  r = g()
end
return r
`, kindRDKTOCLOSE},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ref, _ := luacDump(t, tc.src, false)
			proto, _, err := compiler.Undump(ref, "ref")
			if err != nil {
				t.Fatalf("undump %s chunk: %v", luacBin, err)
			}
			// The case is only meaningful if luac really produced the kind it
			// is named after; otherwise it would pass vacuously against an
			// all-zero field.
			kinds := upvalueKinds(proto)
			found := false
			for _, k := range kinds {
				if k == tc.want {
					found = true
				}
			}
			if !found {
				t.Fatalf("%s emitted no upvalue of kind %d; got kinds %v "+
					"(the source no longer produces the kind this case covers)",
					luacBin, tc.want, kinds)
			}
			// It must come back out byte for byte...
			if got := reDump(t, ref, false); !bytes.Equal(got, ref) {
				t.Errorf("re-dump differs from %s\n%s", luacBin, firstDiff(got, ref))
			}
			// ...and the reloaded prototype must carry the same kinds, not
			// merely happen to produce the same bytes.
			again, _, err := compiler.Undump(reDump(t, ref, false), "again")
			if err != nil {
				t.Fatalf("undump re-dump: %v", err)
			}
			if got := upvalueKinds(again); !bytes.Equal(got, kinds) {
				t.Errorf("upvalue kinds after round-trip: got %v, want %v", got, kinds)
			}
		})
	}
}

// TestUpvalueKindSurvivesStrippedDump checks the same for a stripped chunk. The
// kind byte lives in the upvalue vector, not in the debug information, so
// stripping must leave it alone.
func TestUpvalueKindSurvivesStrippedDump(t *testing.T) {
	requireLuac(t)
	const src = `
local x <const> = {1}
local r
do
  local y <close> = setmetatable({}, {__close = function() end})
  local function g() return x[1], y end
  r = g()
end
return r
`
	for _, strip := range []bool{false, true} {
		ref, _ := luacDump(t, src, strip)
		proto, _, err := compiler.Undump(ref, "ref")
		if err != nil {
			t.Fatalf("strip=%v: undump: %v", strip, err)
		}
		nonZero := false
		for _, k := range upvalueKinds(proto) {
			if k != kindVDKREG {
				nonZero = true
			}
		}
		if !nonZero {
			t.Fatalf("strip=%v: no non-zero upvalue kind to test with", strip)
		}
		if got := reDump(t, ref, strip); !bytes.Equal(got, ref) {
			t.Errorf("strip=%v: re-dump differs\n%s", strip, firstDiff(got, ref))
		}
	}
}

// ---------------------------------------------------------------------------
// OP_SETLIST index boundary
// ---------------------------------------------------------------------------

// refSetList builds a reference-encoded OP_SETLIST. stored is what reference
// puts in vC — the number of elements already in the table — so the batch fills
// indices stored+1 .. stored+n.
func refSetList(a, n, stored, k int) compiler.Instruction {
	return compiler.Instruction(uint32(compiler.OP_SETLIST)<<compiler.PosOP |
		uint32(a)<<compiler.PosA |
		uint32(n)<<compiler.PosVB |
		uint32(stored)<<compiler.PosVC |
		uint32(k)<<compiler.PosK)
}

// goluaSetListOffset decodes the first index a GoLua-encoded OP_SETLIST writes.
// The index is 1-based, so a vC of 0 in the single-word form is free and stands
// for the one index past the field's range.
func goluaSetListOffset(code []compiler.Instruction, i int) int {
	if code[i].K() != 0 {
		return code[i+1].Ax()
	}
	if vc := code[i].VC(); vc != 0 {
		return vc
	}
	return compiler.MaxArgVC + 1
}

func instrEqual(a, b []compiler.Instruction) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestSetListFromRefNeverMistranslates is the direct pin for the boundary.
//
// Reference's lcode.c luaK_setlist uses the single-word form for every
// 'nelems <= MAXARG_vC', so its vC runs the full 0..1023 and the batch it
// describes starts at index 1..1024. GoLua's single-word form keeps the first
// index in that same 10-bit field, and covers the top value because the index
// is 1-based: a vC of 0 means 1024.
//
// The property checked here does not depend on how the field is encoded: for
// every reference encoding, CodeFromRef either produces an instruction that
// writes the batch at exactly the index reference meant, or refuses the chunk.
// It never produces a different index.
func TestSetListFromRefNeverMistranslates(t *testing.T) {
	tail := compiler.ABC(compiler.OP_RETURN0, 0, 0, 0, 0)
	rejected := 0
	for stored := 0; stored <= compiler.MaxArgVC; stored++ {
		code := []compiler.Instruction{refSetList(0, 5, stored, 0), tail}
		if err := compiler.CodeFromRef(code); err != nil {
			rejected++
			continue
		}
		if got, want := goluaSetListOffset(code, 0), stored+1; got != want {
			t.Fatalf("stored=%d: translated to first index %d, want %d", stored, got, want)
		}
		if got := code[0].VB(); got != 5 {
			t.Fatalf("stored=%d: batch size changed to %d", stored, got)
		}
	}
	// The whole single-word range translates. luac emits the top of it —
	// "SETLIST A 1 1023" — for the 1024th element of a constructor written in a
	// register-starved function, so a refusal here would fail a stock chunk.
	if rejected != 0 {
		t.Errorf("single-word range: %d values refused, want none", rejected)
	}

	// The k form spreads the count across vC and the EXTRAARG in units of
	// MaxArgVC+1, and must translate exactly over the whole range that GoLua's
	// own EXTRAARG (25 bits) can name.
	for _, extra := range []int{0, 1, 2, 7, 1000, 32000} {
		for _, vc := range []int{0, 1, compiler.MaxArgVC - 1, compiler.MaxArgVC} {
			stored := vc + extra*(compiler.MaxArgVC+1)
			code := []compiler.Instruction{
				refSetList(0, 3, vc, 1),
				compiler.Ax(compiler.OP_EXTRAARG, extra),
			}
			if err := compiler.CodeFromRef(code); err != nil {
				t.Fatalf("k form vc=%d extra=%d: %v", vc, extra, err)
			}
			if got, want := goluaSetListOffset(code, 0), stored+1; got != want {
				t.Fatalf("k form vc=%d extra=%d: first index %d, want %d", vc, extra, got, want)
			}
		}
	}

	// A k form with no room for its EXTRAARG must be refused, not read past.
	if err := compiler.CodeFromRef([]compiler.Instruction{refSetList(0, 3, 0, 1)}); err == nil {
		t.Errorf("trailing k-form SETLIST accepted with no EXTRAARG after it")
	}
}

// TestSetListRefRoundTripIsExact checks that CodeToRef undoes CodeFromRef on
// every encoding CodeFromRef accepts, so a chunk that loaded can always be
// dumped back to the bytes it came from.
func TestSetListRefRoundTripIsExact(t *testing.T) {
	tail := compiler.ABC(compiler.OP_RETURN0, 0, 0, 0, 0)
	var refs [][]compiler.Instruction
	for stored := 0; stored <= compiler.MaxArgVC; stored++ {
		refs = append(refs, []compiler.Instruction{refSetList(1, stored%64, stored, 0), tail})
	}
	for _, extra := range []int{0, 1, 5, 4095, 32000} {
		for _, vc := range []int{0, 1, 512, compiler.MaxArgVC} {
			refs = append(refs, []compiler.Instruction{
				refSetList(2, 0, vc, 1),
				compiler.Ax(compiler.OP_EXTRAARG, extra),
			})
		}
	}
	for _, want := range refs {
		golua := append([]compiler.Instruction(nil), want...)
		if err := compiler.CodeFromRef(golua); err != nil {
			t.Fatalf("%v: CodeFromRef: %v", want, err)
		}
		if got := compiler.CodeToRef(golua); !instrEqual(got, want) {
			t.Fatalf("round-trip: got %v, want %v", got, want)
		}
	}
}

// listConstructorSource builds a chunk whose table constructor has exactly n
// array elements and which reports enough of the result to catch an off-by-one
// at any batch boundary. Every element's value is its own index, so a batch
// stored at the wrong offset shows up as a wrong value, not merely a wrong
// length. pad extra locals are declared first: reference's lparser.c
// maxtostore() shrinks the SETLIST batch as registers run out, and that is what
// moves the boundary values around.
func listConstructorSource(n, pad int) string {
	var b strings.Builder
	for i := 0; i < pad; i++ {
		fmt.Fprintf(&b, "local pad%d = %d\n", i, i)
	}
	b.WriteString("local t = {")
	for i := 1; i <= n; i++ {
		if i > 1 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, "%d", i)
	}
	b.WriteString("}\n")
	b.WriteString("local bad = 0\n")
	b.WriteString("for i = 1, #t do if t[i] ~= i then bad = bad + 1 end end\n")
	fmt.Fprintf(&b, "print(#t, bad, t[1], t[1022], t[1023], t[1024], t[1025], t[%d], t[%d])\n", n, n+1)
	return b.String()
}

func elemOrNil(i, n int) string {
	if i <= n {
		return fmt.Sprintf("%d", i)
	}
	return "nil"
}

func listConstructorWant(n int) string {
	return fmt.Sprintf("%d\t0\t1\t%s\t%s\t%s\t%s\t%d\tnil",
		n, elemOrNil(1022, n), elemOrNil(1023, n),
		elemOrNil(1024, n), elemOrNil(1025, n), n)
}

// TestSetListConstructorBoundaries runs constructors sized around the encoding
// boundary through luac and back: they must load, execute with every element at
// the index it was written to, and dump back byte for byte.
func TestSetListConstructorBoundaries(t *testing.T) {
	requireLuac(t)
	for _, n := range []int{1022, 1023, 1024, 1025, 1100, 2049} {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			chunk, _ := luacDump(t, listConstructorSource(n, 0), false)
			out := luaOutput(t, "local data = ...\nassert(load(data))()\n",
				vm.NewString(string(chunk)))
			if want := listConstructorWant(n); out != want {
				t.Errorf("running %s chunk: got %q, want %q", luacBin, out, want)
			}
			if got := reDump(t, chunk, false); !bytes.Equal(got, chunk) {
				t.Errorf("re-dump differs\n%s", firstDiff(got, chunk))
			}
		})
	}
}

// hasRefSetListStored reports whether the chunk's raw instruction words contain
// a single-word OP_SETLIST whose vC equals stored. The chunk is scanned as
// bytes rather than loaded, because loading it is what is under test.
func hasRefSetListStored(chunk []byte, stored int) bool {
	for i := 0; i+4 <= len(chunk); i++ {
		w := compiler.Instruction(uint32(chunk[i]) | uint32(chunk[i+1])<<8 |
			uint32(chunk[i+2])<<16 | uint32(chunk[i+3])<<24)
		if w.OpCode() == compiler.OP_SETLIST && w.K() == 0 && w.VC() == stored {
			return true
		}
	}
	return false
}

// TestSetListRefSingleWordBoundaryIsNotCorrupted covers the case that actually
// reaches reference's top single-word value. With enough locals in scope,
// lparser.c maxtostore() drops the batch to one element per SETLIST, so
// 'nelems' steps through every value up to and including MAXARG_vC and luac
// emits "SETLIST A 1 1023" with no k flag — the top of the single-word range,
// which GoLua encodes as a vC of 0.
//
// Such a chunk must load and run with every element at the index reference put
// it, and dump back to the bytes it came from.
func TestSetListRefSingleWordBoundaryIsNotCorrupted(t *testing.T) {
	requireLuac(t)
	// 180 locals leaves fewer than 80 free registers, which is maxtostore()'s
	// one-element-per-flush case.
	const n = 1030
	chunk, _ := luacDump(t, listConstructorSource(n, 180), false)

	// Confirm the chunk really contains the boundary encoding, so this test
	// cannot quietly stop covering it.
	if !hasRefSetListStored(chunk, compiler.MaxArgVC) {
		t.Fatalf("%s emitted no single-word SETLIST with vC == %d; maxtostore() "+
			"behaviour changed and this test no longer covers the boundary",
			luacBin, compiler.MaxArgVC)
	}

	if _, _, err := compiler.Undump(chunk, "boundary"); err != nil {
		t.Fatalf("boundary chunk did not load: %v", err)
	}
	out := luaOutput(t, "local data = ...\nassert(load(data))()\n", vm.NewString(string(chunk)))
	if want := listConstructorWant(n); out != want {
		t.Errorf("boundary chunk ran with wrong contents: got %q, want %q", out, want)
	}
	if got := reDump(t, chunk, false); !bytes.Equal(got, chunk) {
		t.Errorf("re-dump differs\n%s", firstDiff(got, chunk))
	}
}

func hasGoLuaSetListOffset(p *compiler.Proto, offset int) bool {
	for i, inst := range p.Code {
		if inst.OpCode() == compiler.OP_SETLIST && goluaSetListOffset(p.Code, i) == offset {
			return true
		}
	}
	for _, sub := range p.Protos {
		if hasGoLuaSetListOffset(sub, offset) {
			return true
		}
	}
	return false
}

// TestSetListGoLuaBoundaryRoundTrip covers the same boundary from GoLua's own
// side. GoLua's code generator follows the same maxtostore() rule, so with the
// registers this crowded it emits one SETLIST per element and reaches a first
// index of exactly MaxArgVC+1 — the value that needs the k form. Dumping,
// loading and dumping again must reproduce both the bytes and the behaviour.
func TestSetListGoLuaBoundaryRoundTrip(t *testing.T) {
	const n = 1030
	src := listConstructorSource(n, 180)
	want := listConstructorWant(n)

	if out := luaOutput(t, src); out != want {
		t.Fatalf("direct run: got %q, want %q", out, want)
	}
	chunk := goluaDump(t, "@boundary.lua", src, false)
	proto, _, err := compiler.Undump(chunk, "@boundary.lua")
	if err != nil {
		t.Fatalf("undump own dump: %v", err)
	}
	if !hasGoLuaSetListOffset(proto, compiler.MaxArgVC+1) {
		t.Fatalf("GoLua emitted no SETLIST with first index %d; this test no "+
			"longer covers the k-form boundary", compiler.MaxArgVC+1)
	}
	if again := reDump(t, chunk, false); !bytes.Equal(again, chunk) {
		t.Errorf("re-dump differs\n%s", firstDiff(again, chunk))
	}
	if out := luaOutput(t, "local data = ...\nassert(load(data))()\n",
		vm.NewString(string(chunk))); out != want {
		t.Errorf("after round-trip: got %q, want %q", out, want)
	}
}

// ---------------------------------------------------------------------------
// Vararg register bound
// ---------------------------------------------------------------------------

// Prototype flag bits, as the test builds them by hand.
const (
	pfVaHidBit = 1
	pfVaTabBit = 2
)

// chunkWithVarargFlag builds a minimal, otherwise well-formed chunk whose only
// function carries the given flag byte, parameter count and stack size.
func chunkWithVarargFlag(flag, numparams, maxstack byte) []byte {
	var buf bytes.Buffer
	buf.Write(validChunkHeader())
	putVarint(&buf, 0) // linedefined
	putVarint(&buf, 0) // lastlinedefined
	buf.WriteByte(numparams)
	buf.WriteByte(flag)
	buf.WriteByte(maxstack)
	putVarint(&buf, 0) // sizecode
	for buf.Len()%4 != 0 {
		buf.WriteByte(0)
	}
	putVarint(&buf, 0) // sizek
	putVarint(&buf, 0) // sizeupvalues
	putVarint(&buf, 0) // sizep
	putVarint(&buf, 0) // source: "no string" marker
	putVarint(&buf, 0)
	putVarint(&buf, 0) // sizelineinfo
	putVarint(&buf, 0) // sizeabslineinfo
	putVarint(&buf, 0) // sizelocvars
	putVarint(&buf, 0) // sizeupvalnames
	return buf.Bytes()
}

// TestVarargRegisterOutOfFrameIsRejected pins the hardening on the flag byte.
// Both vararg forms make the VM write the register just past the fixed
// parameters — the vararg table for PF_VATAB, a nil for PF_VAHID — and it does
// so with no bound of its own. Reference always reserves that register (its
// parlist creates a local there and then reserves registers for it), so
// numparams is always below maxstacksize for a vararg prototype; a chunk that
// says otherwise is corrupt and must be refused at load time rather than
// allowed to write outside the frame.
func TestVarargRegisterOutOfFrameIsRejected(t *testing.T) {
	for _, flag := range []byte{pfVaHidBit, pfVaTabBit} {
		for _, np := range []byte{0, 2, 3, 40, 255} {
			// maxstacksize == numparams: the vararg slot would be the first
			// register past the frame.
			_, _, err := compiler.Undump(chunkWithVarargFlag(flag, np, np), "evil")
			if err == nil {
				t.Errorf("flag=%d numparams=%d: loaded with the vararg slot outside the frame",
					flag, np)
				continue
			}
			if !strings.Contains(err.Error(), "bad binary format") {
				t.Errorf("flag=%d numparams=%d: unexpected error %v", flag, np, err)
			}
		}
		// One register beyond the parameters is the reference layout and must
		// still load.
		if _, _, err := compiler.Undump(chunkWithVarargFlag(flag, 2, 3), "ok"); err != nil {
			t.Errorf("flag=%d: valid vararg prototype rejected: %v", flag, err)
		}
	}
	// A non-vararg prototype is unaffected: it owns no slot past its
	// parameters, so numparams == maxstacksize is none of this rule's business.
	if _, _, err := compiler.Undump(chunkWithVarargFlag(0, 2, 2), "plain"); err != nil {
		t.Errorf("non-vararg prototype rejected: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Named vararg coverage, stated honestly
// ---------------------------------------------------------------------------

func hasNamedVararg(p *compiler.Proto) bool {
	if p.HasNamedVarArg {
		return true
	}
	for _, sub := range p.Protos {
		if hasNamedVararg(sub) {
			return true
		}
	}
	return false
}

func hasOpcode(p *compiler.Proto, op compiler.OpCode) bool {
	for _, inst := range p.Code {
		if inst.OpCode() == op {
			return true
		}
	}
	for _, sub := range p.Protos {
		if hasOpcode(sub, op) {
			return true
		}
	}
	return false
}

// TestNamedVarargBothForms covers the two shapes reference gives a named
// vararg parameter, which are not interchangeable in the chunk.
//
// lparser.c setvararg marks every vararg function PF_VAHID; only lcode.c
// needvatab promotes it to PF_VATAB, and that happens when the vararg parameter
// is used as a plain value (bound to a local, captured as an upvalue, assigned
// through). A function that merely indexes its named vararg —
// "local function f(a, ... args) return args[1] end", the ordinary way to write
// it — stays PF_VAHID: the extra arguments are never collected into a table and
// the reads go straight to them, through OP_GETVARG.
func TestNamedVarargBothForms(t *testing.T) {
	requireLuac(t)

	// The table form: needvatab fires, and GoLua runs it.
	chunk, _ := luacDump(t, `
local function f(a, ... args)
  local t = args
  return a, t.n, t[1], select("#", ...)
end
print(f(1, 2, 3))
`, false)
	proto, _, err := compiler.Undump(chunk, "vatab")
	if err != nil {
		t.Fatalf("undump table form: %v", err)
	}
	if !hasNamedVararg(proto) {
		t.Fatalf("expected a PF_VATAB prototype in the table form")
	}
	if out := luaOutput(t, "local data = ...\nassert(load(data))()\n",
		vm.NewString(string(chunk))); out != "1\t2\t2\t2" {
		t.Errorf("PF_VATAB named vararg: got %q, want %q", out, "1\t2\t2\t2")
	}

	// The hidden form: same shape of source, but the vararg is only indexed, so
	// reference leaves it PF_VAHID and emits OP_GETVARG.
	chunk, _ = luacDump(t, `
local function f(a, ... args)
  return a, args[1], args.n
end
print(f(1, 2, 3))
`, false)
	proto, _, err = compiler.Undump(chunk, "vahid")
	if err != nil {
		t.Fatalf("undump hidden form: %v", err)
	}
	if hasNamedVararg(proto) {
		t.Fatalf("expected no PF_VATAB prototype in the hidden form; " +
			"reference's needvatab rule changed")
	}
	if !hasOpcode(proto, compiler.OP_GETVARG) {
		t.Fatalf("expected OP_GETVARG in the hidden form")
	}
	if out := luaOutput(t, "local data = ...\nassert(load(data))()\n",
		vm.NewString(string(chunk))); out != "1\t2\t2" {
		t.Errorf("PF_VAHID named vararg: got %q, want %q", out, "1\t2\t2")
	}

	// The hidden form's reads answer only an integral index in range and the
	// key "n"; a string that merely looks like a number is not coerced, and
	// nothing else names an argument (ltm.c luaT_getvararg).
	chunk, _ = luacDump(t, `
local function f(... args)
  return args[1], args["1"], args[1.0], args[1.5], args[0], args[4], args.n, args.x
end
print(f("a", "b", "c"))
`, false)
	if out := luaOutput(t, "local data = ...\nassert(load(data))()\n",
		vm.NewString(string(chunk))); out != "a\tnil\ta\tnil\tnil\tnil\t3\tnil" {
		t.Errorf("PF_VAHID vararg indexing: got %q, want %q", out,
			"a\tnil\ta\tnil\tnil\tnil\t3\tnil")
	}
}
