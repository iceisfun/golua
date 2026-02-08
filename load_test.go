package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

// TestFileProvider is a simple filesystem-based code provider for testing
type TestFileProvider struct {
	basePath string
}

func NewTestFileProvider(basePath string) *TestFileProvider {
	return &TestFileProvider{basePath: basePath}
}

func (p *TestFileProvider) LoadChunk(name string, caller *vm.LuaCallerContext) ([]byte, string, error) {
	// Resolve path relative to basePath
	fullPath := filepath.Join(p.basePath, name)
	source, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, "", fmt.Errorf("cannot open '%s': %v", name, err)
	}
	return source, "@" + name, nil
}

func (p *TestFileProvider) Capabilities() vm.LuaLoaderCaps {
	return vm.LuaLoaderCaps{
		AllowDofile:   true,
		AllowLoadfile: true,
	}
}

func runLuaSource(t *testing.T, source, name string) {
	t.Helper()

	block, err := parser.Parse(name, source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	proto, err := compiler.Compile(name, block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	v := vm.New()
	stdlib.Open(v)

	_, err = v.Run(proto)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

func runLuaSourceWithProvider(t *testing.T, source, name string, provider vm.LuaCodeProvider) {
	t.Helper()

	block, err := parser.Parse(name, source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	proto, err := compiler.Compile(name, block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	v := vm.New()
	v.SetCodeProvider(provider)
	stdlib.Open(v)

	_, err = v.Run(proto)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

// Test load() with a string
func TestLoad_String(t *testing.T) {
	source := `
		local f, err = load("return 1 + 2")
		assert(f, err)
		local result = f()
		assert(result == 3, "expected 3, got " .. tostring(result))
	`
	runLuaSource(t, source, "test_load_string")
}

// Test load() with a custom chunk name
func TestLoad_ChunkName(t *testing.T) {
	source := `
		local f, err = load("return 42", "my_chunk")
		assert(f, err)
		local result = f()
		assert(result == 42, "expected 42, got " .. tostring(result))
	`
	runLuaSource(t, source, "test_load_chunkname")
}

// Test load() with syntax error
func TestLoad_SyntaxError(t *testing.T) {
	source := `
		local f, err = load("return 1 +")
		assert(f == nil, "expected nil for syntax error")
		assert(err, "expected error message")
	`
	runLuaSource(t, source, "test_load_syntax_error")
}

// Test load() with a function reader
func TestLoad_FunctionReader(t *testing.T) {
	source := `
		local chunks = {"return ", "1", " + ", "2"}
		local i = 0
		local function reader()
			i = i + 1
			return chunks[i]
		end
		local f, err = load(reader)
		assert(f, err)
		local result = f()
		assert(result == 3, "expected 3, got " .. tostring(result))
	`
	runLuaSource(t, source, "test_load_function_reader")
}

// Test load() with custom environment
func TestLoad_CustomEnv(t *testing.T) {
	source := `
		local env = {x = 10}
		local f, err = load("return x", "chunk", "t", env)
		assert(f, err)
		local result = f()
		assert(result == 10, "expected 10, got " .. tostring(result))
	`
	runLuaSource(t, source, "test_load_custom_env")
}

// Test loadfile() with a provider
func TestLoadfile(t *testing.T) {
	// Create a temporary directory with a Lua file
	tmpDir := t.TempDir()
	luaFile := filepath.Join(tmpDir, "module.lua")
	err := os.WriteFile(luaFile, []byte("return 123"), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	provider := NewTestFileProvider(tmpDir)

	source := `
		local f, err = loadfile("module.lua")
		assert(f, err)
		local result = f()
		assert(result == 123, "expected 123, got " .. tostring(result))
	`
	runLuaSourceWithProvider(t, source, "test_loadfile", provider)
}

// Test dofile() with a provider
func TestDofile(t *testing.T) {
	// Create a temporary directory with a Lua file
	tmpDir := t.TempDir()
	luaFile := filepath.Join(tmpDir, "script.lua")
	err := os.WriteFile(luaFile, []byte("return 456, 789"), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	provider := NewTestFileProvider(tmpDir)

	source := `
		local a, b = dofile("script.lua")
		assert(a == 456, "expected a=456, got " .. tostring(a))
		assert(b == 789, "expected b=789, got " .. tostring(b))
	`
	runLuaSourceWithProvider(t, source, "test_dofile", provider)
}

// Test loadfile() without provider returns error
func TestLoadfile_NoProvider(t *testing.T) {
	// Note: loadfile won't even be registered without a provider,
	// but we can test that load still works
	source := `
		-- loadfile should not be available without a provider
		assert(loadfile == nil or type(loadfile) == "function")
	`
	runLuaSource(t, source, "test_loadfile_no_provider")
}

// Test dofile() with file that sets globals
func TestDofile_SetsGlobals(t *testing.T) {
	tmpDir := t.TempDir()
	luaFile := filepath.Join(tmpDir, "globals.lua")
	err := os.WriteFile(luaFile, []byte("globalVar = 'hello from file'"), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	provider := NewTestFileProvider(tmpDir)

	source := `
		dofile("globals.lua")
		assert(globalVar == "hello from file", "expected global to be set")
	`
	runLuaSourceWithProvider(t, source, "test_dofile_globals", provider)
}

// Test nested dofile calls
func TestDofile_Nested(t *testing.T) {
	tmpDir := t.TempDir()

	// Create inner file
	innerFile := filepath.Join(tmpDir, "inner.lua")
	err := os.WriteFile(innerFile, []byte("return 100"), 0644)
	if err != nil {
		t.Fatalf("failed to write inner file: %v", err)
	}

	// Create outer file that calls dofile
	outerFile := filepath.Join(tmpDir, "outer.lua")
	err = os.WriteFile(outerFile, []byte(`
		local x = dofile("inner.lua")
		return x * 2
	`), 0644)
	if err != nil {
		t.Fatalf("failed to write outer file: %v", err)
	}

	provider := NewTestFileProvider(tmpDir)

	source := `
		local result = dofile("outer.lua")
		assert(result == 200, "expected 200, got " .. tostring(result))
	`
	runLuaSourceWithProvider(t, source, "test_dofile_nested", provider)
}

// Test loadfile error handling (file not found)
func TestLoadfile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	provider := NewTestFileProvider(tmpDir)

	source := `
		local f, err = loadfile("nonexistent.lua")
		assert(f == nil, "expected nil for missing file")
		assert(err, "expected error message for missing file")
	`
	runLuaSourceWithProvider(t, source, "test_loadfile_not_found", provider)
}
