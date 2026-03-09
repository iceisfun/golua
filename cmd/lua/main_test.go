package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempLua(t *testing.T, dir, name, source string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("write temp lua: %v", err)
	}
	return path
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
	if !strings.Contains(msg, "[C]: in function 'math.abs'") {
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
	if !strings.Contains(msg, "[C]: in function 'error'") {
		t.Fatalf("expected outer traceback for hook error, got: %s", msg)
	}
}
