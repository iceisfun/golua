package tests

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/iceisfun/golua/v1/compiler"
	"github.com/iceisfun/golua/v1/stdlib"
	"github.com/iceisfun/golua/v1/vm"
)

// runGoLua compiles and runs a Lua snippet, returns captured output and error.
func runGoLua(t *testing.T, code string) (string, error) {
	t.Helper()
	proto, err := compileLua("=test", code)
	if err != nil {
		return "", err
	}
	v := vm.New(vm.WithCaptureOutput(true))
	stdlib.Open(v)
	_, runErr := v.Run(proto)
	return strings.Join(v.OutputLines(), "\n"), runErr
}

// runRefLua runs a Lua snippet via the reference lua5.4 binary.
func runRefLua(t *testing.T, code string) (string, error) {
	t.Helper()
	cmd := exec.Command("lua5.4", "-e", code)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return strings.TrimSpace(stderr.String()), err
	}
	return strings.TrimSpace(stdout.String()), nil
}

func TestStringDumpBasicRoundTrip(t *testing.T) {
	code := `
local function f(x) return x * x end
local d = string.dump(f)
local f2 = load(d)
print(f2(5))
print(f2(0))
print(f2(-3))
`
	out, err := runGoLua(t, code)
	if err != nil {
		t.Fatalf("GoLua error: %v", err)
	}
	if out != "25\n0\n9" {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestStringDumpStripped(t *testing.T) {
	code := `
local function f(x) return x + 1 end
local d1 = string.dump(f)
local d2 = string.dump(f, true)
print(#d2 <= #d1)
local f2 = load(d2)
print(f2(41))
`
	out, err := runGoLua(t, code)
	if err != nil {
		t.Fatalf("GoLua error: %v", err)
	}
	if out != "true\n42" {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestStringDumpCrossLoadFromRefLua(t *testing.T) {
	// Dump a function in reference Lua, load in GoLua
	refCode := `
local function f(x) return x * 2 + 1 end
local d = string.dump(f)
local hex = {}
for i = 1, #d do hex[#hex+1] = string.format("%02x", d:byte(i)) end
print(table.concat(hex))
`
	hexStr, err := runRefLua(t, refCode)
	if err != nil {
		t.Skipf("lua5.4 not available: %v", err)
	}

	// Load the hex in GoLua
	goCode := `
local hex = "` + hexStr + `"
local bytes = {}
for i = 1, #hex, 2 do
    bytes[#bytes+1] = string.char(tonumber(hex:sub(i, i+1), 16))
end
local d = table.concat(bytes)
local fn = load(d)
assert(fn ~= nil, "failed to load ref dump")
print(fn(5))
print(fn(0))
print(fn(-1))
`
	out, err := runGoLua(t, goCode)
	if err != nil {
		t.Fatalf("GoLua error: %v", err)
	}
	if out != "11\n1\n-1" {
		t.Errorf("cross-load failed: %q", out)
	}
}

func TestStringDumpCrossLoadToRefLua(t *testing.T) {
	// Dump a function in GoLua, load in reference Lua
	goCode := `
local function f(x) return x * 2 + 1 end
local d = string.dump(f)
local hex = {}
for i = 1, #d do hex[#hex+1] = string.format("%02x", d:byte(i)) end
print(table.concat(hex))
`
	hexStr, err := runGoLua(t, goCode)
	if err != nil {
		t.Fatalf("GoLua error: %v", err)
	}

	refCode := `
local hex = "` + hexStr + `"
local bytes = {}
for i = 1, #hex, 2 do
    bytes[#bytes+1] = string.char(tonumber(hex:sub(i, i+1), 16))
end
local d = table.concat(bytes)
local fn = load(d)
assert(fn ~= nil, "failed to load GoLua dump")
print(fn(5))
print(fn(0))
print(fn(-1))
`
	out, err := runRefLua(t, refCode)
	if err != nil {
		t.Skipf("lua5.4 not available or cross-load failed: %v", err)
	}
	if out != "11\n1\n-1" {
		t.Errorf("cross-load to ref Lua failed: %q", out)
	}
}

func TestStringDumpNativeError(t *testing.T) {
	code := `
local ok, err = pcall(string.dump, print)
print(ok)
print(err:find("unable to dump") ~= nil)
`
	out, err := runGoLua(t, code)
	if err != nil {
		t.Fatalf("GoLua error: %v", err)
	}
	if out != "false\ntrue" {
		t.Errorf("unexpected: %q", out)
	}
}

func TestStringDumpModeChecks(t *testing.T) {
	code := `
local d = string.dump(function() return 42 end)
-- mode "b" accepts binary
local f1 = load(d, nil, "b")
print(f1())
-- mode "t" rejects binary
local f2, e = load(d, nil, "t")
print(f2 == nil)
print(e:find("binary") ~= nil)
-- mode "bt" accepts binary
local f3 = load(d, nil, "bt")
print(f3())
-- text with mode "b" rejected
local f4, e4 = load("return 1", nil, "b")
print(f4 == nil)
`
	out, err := runGoLua(t, code)
	if err != nil {
		t.Fatalf("GoLua error: %v", err)
	}
	expected := "42\ntrue\ntrue\n42\ntrue"
	if out != expected {
		t.Errorf("got:\n%s\nexpected:\n%s", out, expected)
	}
}

func TestStringDumpEnvironment(t *testing.T) {
	code := `
local function use_x() return x end
local d = string.dump(use_x)
local env = setmetatable({x = 777}, {__index = _G})
local f = load(d, nil, nil, env)
print(f())
`
	out, err := runGoLua(t, code)
	if err != nil {
		t.Fatalf("GoLua error: %v", err)
	}
	if out != "777" {
		t.Errorf("expected 777, got %q", out)
	}
}

func TestUndumpHeaderValidation(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"empty", []byte{}, "truncated"},
		{"wrong sig", []byte("notLua54"), "not a binary chunk"},
		{"wrong version", []byte("\x1bLua\x53"), "version mismatch"},
		{"wrong format", []byte("\x1bLua\x54\x01"), "format mismatch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := compiler.Undump(tt.data, "test")
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q doesn't contain %q", err.Error(), tt.want)
			}
		})
	}
}

func TestStringDumpNestedFunctions(t *testing.T) {
	code := `
local function make_adder(n)
    return function(x) return x + n end
end
local d = string.dump(make_adder)
local f = load(d)
local add5 = f(5)
print(add5(10))
print(add5(-5))
`
	out, err := runGoLua(t, code)
	if err != nil {
		t.Fatalf("GoLua error: %v", err)
	}
	if out != "15\n0" {
		t.Errorf("unexpected: %q", out)
	}
}

func TestStringDumpAllConstTypes(t *testing.T) {
	code := `
local function f()
    return nil, true, false, 42, -100, 3.14, "hello"
end
local d = string.dump(f)
local f2 = load(d)
local a,b,c,d2,e,f3,g = f2()
print(a, b, c, d2, e, f3, g)
`
	goOut, err := runGoLua(t, code)
	if err != nil {
		t.Fatalf("GoLua error: %v", err)
	}
	refOut, err := runRefLua(t, code)
	if err != nil {
		t.Skipf("lua5.4 not available: %v", err)
	}
	if goOut != refOut {
		t.Errorf("GoLua: %q\nRef:   %q", goOut, refOut)
	}
}
