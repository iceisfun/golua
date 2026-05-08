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
