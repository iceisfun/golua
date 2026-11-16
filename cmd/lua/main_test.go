package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

func writeTempLua(t *testing.T, dir, name, source string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("write temp lua: %v", err)
	}
	return path
}

func runLuaWithCLIProviders(t *testing.T, testMode bool, scriptDir, source string) ([]vm.Value, error) {
	t.Helper()

	block, err := parser.Parse("test", source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	proto, err := compiler.Compile("=test", block)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	v := vm.New()
	v.SetOsProvider(vm.NewDefaultOsProvider())
	configureCLIProviders(v, testMode, scriptDir)
	stdlib.Open(v)
	return v.Run(proto)
}

func TestRunCLI_TopLevelNativeErrorIncludesTraceback(t *testing.T) {
	dir := t.TempDir()
	script := writeTempLua(t, dir, "bad_abs.lua", "math.abs()\n")

	var stderr bytes.Buffer
	exitCode := runCLI([]string{"lua", script}, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}

	msg := stderr.String()
	if !strings.Contains(msg, "lua: ") {
		t.Fatalf("expected program prefix, got: %s", msg)
	}
	if !strings.Contains(msg, "[C]: in field 'abs'") {
		t.Fatalf("expected native traceback frame, got: %s", msg)
	}
	if !strings.Contains(msg, "in main chunk") {
		t.Fatalf("expected main chunk frame, got: %s", msg)
	}
}

func TestRunCLI_HookErrorIncludesHookTraceback(t *testing.T) {
	dir := t.TempDir()
	source := "debug.sethook(function() error(debug.traceback('hookboom', 0)) end, 'c')\npcall(function() end)\n"
	script := writeTempLua(t, dir, "hook_err.lua", source)

	var stderr bytes.Buffer
	exitCode := runCLI([]string{"lua", script}, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}

	msg := stderr.String()
	if !strings.Contains(msg, "in hook '?'") {
		t.Fatalf("expected hook frame, got: %s", msg)
	}
	if !strings.Contains(msg, "[C]: in global 'error'") {
		t.Fatalf("expected outer traceback for hook error, got: %s", msg)
	}
}

// TestRunCLI_TostringErrorShowsCombinedTraceback verifies that when
// error(obj) is called with a table whose __tostring itself errors,
// the output includes both the inner __tostring traceback AND the
// outer error() site, matching C Lua's combined traceback.
func TestRunCLI_TostringErrorShowsCombinedTraceback(t *testing.T) {
	dir := t.TempDir()
	source := `local mt = {
  __tostring = function()
    error("inner")
  end
}
error(setmetatable({}, mt))
`
	script := writeTempLua(t, dir, "tostring_err.lua", source)

	var stderr bytes.Buffer
	exitCode := runCLI([]string{"lua", script}, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}

	msg := stderr.String()
	// Inner error message should be present
	if !strings.Contains(msg, "inner") {
		t.Fatalf("expected 'inner' error message, got: %s", msg)
	}
	// Inner __tostring frame
	if !strings.Contains(msg, "in function <") {
		t.Fatalf("expected inner __tostring frame, got: %s", msg)
	}
	// Outer error() call site from main chunk
	if !strings.Contains(msg, "in main chunk") {
		t.Fatalf("expected outer 'in main chunk' frame, got: %s", msg)
	}
}

// TestRunCLI_TransformedErrorTostringPreservesInnerTraceback verifies that a
// transformed error object returned from xpcall still carries the inner
// __tostring failure traceback when later stringified at the CLI top level.
func TestRunCLI_TransformedErrorTostringPreservesInnerTraceback(t *testing.T) {
	dir := t.TempDir()
	source := `local ok, err = xpcall(function()
  error(setmetatable({}, {
    __tostring = function()
      error("inner")
    end,
  }))
end, function(e)
  return e
end)
print(ok)
print(tostring(err))
`
	script := writeTempLua(t, dir, "transformed_tostring_err.lua", source)

	var stderr bytes.Buffer
	exitCode := runCLI([]string{"lua", script}, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}

	msg := stderr.String()
	if !strings.Contains(msg, "inner") {
		t.Fatalf("expected inner error message, got: %s", msg)
	}
	if !strings.Contains(msg, "[C]: in global 'error'") {
		t.Fatalf("expected inner error frame, got: %s", msg)
	}
	if !strings.Contains(msg, "in function <") {
		t.Fatalf("expected inner __tostring frame, got: %s", msg)
	}
	if !strings.Contains(msg, "[C]: in global 'tostring'") {
		t.Fatalf("expected outer tostring frame, got: %s", msg)
	}
}

func TestConfigureCLIProviders_TestModeUsesJailedIO(t *testing.T) {
	dir := t.TempDir()
	results, err := runLuaWithCLIProviders(t, true, dir, `
		local f, openErr = io.open("probe.txt", "w")
		return f == nil, type(openErr) == "string"
	`)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if len(results) != 2 || !results[0].AsBool() || !results[1].AsBool() {
		t.Fatalf("unexpected results: %#v", results)
	}
}

func TestConfigureCLIProviders_DefaultModeAllowsWrite(t *testing.T) {
	dir := t.TempDir()
	results, err := runLuaWithCLIProviders(t, false, dir, `
		local f, openErr = io.open("probe.txt", "w")
		if not f then
			return false, openErr
		end
		f:write("ok")
		f:close()
		return true, io.open("probe.txt", "r") ~= nil
	`)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if len(results) != 2 || !results[0].AsBool() || !results[1].AsBool() {
		t.Fatalf("unexpected results: %#v", results)
	}
	if _, err := os.Stat(filepath.Join(dir, "probe.txt")); err != nil {
		t.Fatalf("expected written file: %v", err)
	}
}

func TestConfigureCLIProviders_DefaultModeEnablesPopen(t *testing.T) {
	dir := t.TempDir()
	// io.popen should be available (and functional) in default CLI mode, matching
	// reference standalone Lua which exposes both os.execute and io.popen.
	results, err := runLuaWithCLIProviders(t, false, dir, `
		if type(io.popen) ~= "function" then
			return false, "io.popen not a function: " .. type(io.popen)
		end
		local f, openErr = io.popen("echo hello")
		if not f then
			return false, "popen failed: " .. tostring(openErr)
		end
		local out = f:read("a")
		f:close()
		return out == "hello\n", out
	`)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if len(results) < 1 || !results[0].AsBool() {
		t.Fatalf("io.popen did not work as expected: %#v", results)
	}
}

func TestConfigureCLIProviders_TestModeNoPopen(t *testing.T) {
	dir := t.TempDir()
	// Jailed/test mode must NOT expose io.popen (no process provider set).
	results, err := runLuaWithCLIProviders(t, true, dir, `
		return io.popen == nil
	`)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if len(results) != 1 || !results[0].AsBool() {
		t.Fatalf("expected io.popen to be nil in test mode: %#v", results)
	}
}

func TestRunCLI_DoubleDashStopsOptionParsing(t *testing.T) {
	dir := t.TempDir()
	script := writeTempLua(t, dir, "-e", `print("ok")`)

	var stderr bytes.Buffer
	exitCode := runCLI([]string{"lua", "--", script}, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %q", stderr.String())
	}
}

func TestRunCLI_BinaryChunkAfterBOMOrShebang(t *testing.T) {
	res, err := runLuaWithCLIProviders(t, false, "", `return string.dump(function() os.exit(7) end)`)
	if err != nil || len(res) != 1 {
		t.Fatalf("dump: %v %v", res, err)
	}
	dump := res[0].AsString()

	for name, prefix := range map[string]string{
		"bom":         "\xEF\xBB\xBF",
		"shebang":     "#!x\n",
		"bom+shebang": "\xEF\xBB\xBF#!x\n",
	} {
		script := writeTempLua(t, t.TempDir(), "chunk.luac", prefix+dump)
		var stderr bytes.Buffer
		if code := runCLI([]string{"lua", script}, &stderr); code != 7 {
			t.Errorf("%s: exit code = %d, want 7; stderr=%q", name, code, stderr.String())
		}
	}
}

// Only file sources skip a BOM and '#' line before the binary check, like
// luaL_loadfilex. -e and LUA_INIT strings load like lua.c's dostring: text
// only, parsed exactly as given.
func TestRunCLI_StringChunksAreTextOnly(t *testing.T) {
	t.Setenv("LUA_INIT_5_5", "")
	os.Unsetenv("LUA_INIT_5_5")
	t.Setenv("LUA_INIT", "")
	os.Unsetenv("LUA_INIT")
	script := writeTempLua(t, t.TempDir(), "m.lua", "os.exit(5)")

	cases := []struct {
		name    string
		argv    []string
		init    string
		wantErr string
	}{
		{"e shebang", []string{"lua", "-e", "#x"}, "", "(command line):1: unexpected symbol near '#'"},
		{"e binary", []string{"lua", "-e", "\x1bLuaX"}, "", "attempt to load a binary chunk (mode is 't')"},
		{"e bom", []string{"lua", "-e", "\xEF\xBB\xBFx=1"}, "", "(command line):1: unexpected symbol near '<\\239>'"},
		{"init shebang binary", []string{"lua", script}, "#!x\n\x1bLua", "LUA_INIT:1: unexpected symbol near '#'"},
		{"init binary", []string{"lua", script}, "\x1bLuaX", "attempt to load a binary chunk (mode is 't')"},
	}
	for _, c := range cases {
		if c.init != "" {
			os.Setenv("LUA_INIT", c.init)
		} else {
			os.Unsetenv("LUA_INIT")
		}
		var stderr bytes.Buffer
		code := runCLI(c.argv, &stderr)
		if code != 1 || !strings.Contains(stderr.String(), c.wantErr) {
			t.Errorf("%s: code=%d stderr=%q, want 1 and %q", c.name, code, stderr.String(), c.wantErr)
		}
	}
}

// A file whose first byte (after any BOM/'#' prefix) is ESC goes to the
// undumper, and its errors name the file without the '@' prefix.
func TestRunCLI_BadBinaryScriptMessage(t *testing.T) {
	dir := t.TempDir()
	for name, tc := range map[string]struct{ src, want string }{
		"bad signature":    {"\x1bXYZ", "bad binary format (not a binary chunk)"},
		"prefixed garbage": {"#!x\n\x1bLuaGARBAGE", "bad binary format (version mismatch)"},
	} {
		script := writeTempLua(t, dir, "b.lua", tc.src)
		var stderr bytes.Buffer
		code := runCLI([]string{"lua", script}, &stderr)
		want := "lua: " + script + ": " + tc.want
		if code != 1 || !strings.Contains(stderr.String(), want) {
			t.Errorf("%s: code=%d stderr=%q, want %q", name, code, stderr.String(), want)
		}
	}
}
