package golua_test

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

func TestLoad_ReaderNonStringTopLevelTraceback(t *testing.T) {
	source := `
		local n = 0
		local f, err = load(function()
			n = n + 1
			if n == 1 then
				return {}
			end
			return nil
		end)
		assert(f == nil, "expected load to fail")
		assert(type(err) == "string", type(err))
		assert(err:find("reader function must return a string", 1, true), err)
		assert(err:find("stack traceback:", 1, true), err)
		assert(err:find("[C]: in function 'load'", 1, true), err)
	`
	runLuaSource(t, source, "test_load_reader_nonstring_top_level_traceback")
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

// MockProvider is an in-memory code provider for testing dofile without filesystem access.
type MockProvider struct {
	files map[string]string
	caps  vm.LuaLoaderCaps
}

func (p *MockProvider) LoadChunk(name string, caller *vm.LuaCallerContext) ([]byte, string, error) {
	src, ok := p.files[name]
	if !ok {
		return nil, "", fmt.Errorf("cannot open '%s': no such file", name)
	}
	return []byte(src), "@" + name, nil
}

func (p *MockProvider) Capabilities() vm.LuaLoaderCaps {
	return p.caps
}

func newMockProvider(files map[string]string) *MockProvider {
	return &MockProvider{
		files: files,
		caps:  vm.LuaLoaderCaps{AllowDofile: true, AllowLoadfile: true},
	}
}

func TestDofile_MockProvider(t *testing.T) {
	provider := newMockProvider(map[string]string{
		"hello.lua": `return "hello"`,
	})
	source := `
		local r = dofile("hello.lua")
		assert(r == "hello", "expected 'hello', got " .. tostring(r))
	`
	runLuaSourceWithProvider(t, source, "test", provider)
}

func TestDofile_MultipleReturnsMock(t *testing.T) {
	provider := newMockProvider(map[string]string{
		"multi.lua": `return 1, 2, 3`,
	})
	source := `
		local a, b, c = dofile("multi.lua")
		assert(a == 1 and b == 2 and c == 3,
			string.format("expected 1,2,3 got %s,%s,%s", tostring(a), tostring(b), tostring(c)))
	`
	runLuaSourceWithProvider(t, source, "test", provider)
}

func TestDofile_RuntimeError(t *testing.T) {
	provider := newMockProvider(map[string]string{
		"err.lua": `error("boom")`,
	})
	source := `
		local ok, err = pcall(dofile, "err.lua")
		assert(not ok, "expected failure")
		assert(type(err) == "string", "expected string error, got " .. type(err))
		assert(err:find("boom"), "expected 'boom' in error: " .. tostring(err))
	`
	runLuaSourceWithProvider(t, source, "test", provider)
}

func TestDofile_ErrorValuePreserved(t *testing.T) {
	provider := newMockProvider(map[string]string{
		"tbl_err.lua": `error({msg="bad"})`,
	})
	source := `
		local ok, err = pcall(dofile, "tbl_err.lua")
		assert(not ok, "expected failure")
		assert(type(err) == "table", "expected table error, got " .. type(err))
		assert(err.msg == "bad", "expected err.msg=='bad', got " .. tostring(err.msg))
	`
	runLuaSourceWithProvider(t, source, "test", provider)
}

func TestDofile_ProviderError(t *testing.T) {
	provider := newMockProvider(map[string]string{}) // no files
	source := `
		local ok, err = pcall(dofile, "missing.lua")
		assert(not ok, "expected failure")
		assert(type(err) == "string", "expected string error")
		assert(err:find("cannot open"), "expected 'cannot open' in: " .. tostring(err))
	`
	runLuaSourceWithProvider(t, source, "test", provider)
}

func TestDofile_SyntaxError(t *testing.T) {
	provider := newMockProvider(map[string]string{
		"bad.lua": `return 1 +`,
	})
	source := `
		local ok, err = pcall(dofile, "bad.lua")
		assert(not ok, "expected failure")
		assert(type(err) == "string", "expected string error")
	`
	runLuaSourceWithProvider(t, source, "test", provider)
}

func TestDofile_NoProvider(t *testing.T) {
	// Without a provider, dofile should not be registered
	source := `
		assert(dofile == nil, "expected dofile to be nil without provider")
	`
	runLuaSource(t, source, "test")
}

func TestDofile_NotPermitted(t *testing.T) {
	provider := &MockProvider{
		files: map[string]string{"x.lua": "return 1"},
		caps:  vm.LuaLoaderCaps{AllowDofile: false, AllowLoadfile: true},
	}
	source := `
		assert(dofile == nil, "expected dofile to be nil when not permitted")
	`
	runLuaSourceWithProvider(t, source, "test", provider)
}

func TestDofile_NestedMock(t *testing.T) {
	provider := newMockProvider(map[string]string{
		"inner.lua": `return 10`,
		"outer.lua": `return dofile("inner.lua") * 3`,
	})
	source := `
		local r = dofile("outer.lua")
		assert(r == 30, "expected 30, got " .. tostring(r))
	`
	runLuaSourceWithProvider(t, source, "test", provider)
}

func TestDofile_ProviderReceivesPath(t *testing.T) {
	// Verify the provider receives the exact filename argument
	provider := newMockProvider(map[string]string{
		"some/path/file.lua": `return "ok"`,
	})
	source := `
		local r = dofile("some/path/file.lua")
		assert(r == "ok", "expected 'ok', got " .. tostring(r))
	`
	runLuaSourceWithProvider(t, source, "test", provider)
}

func TestDofile_Shebang(t *testing.T) {
	provider := newMockProvider(map[string]string{
		"shebang.lua": "#!/usr/bin/lua\nreturn 42",
	})
	source := `
		local r = dofile("shebang.lua")
		assert(r == 42, "expected 42, got " .. tostring(r))
	`
	runLuaSourceWithProvider(t, source, "test", provider)
}

func TestLoadfile_NumberFilenameCoercion(t *testing.T) {
	provider := newMockProvider(map[string]string{
		"123": `return "num-ok"`,
	})
	source := `
		local f, err = loadfile(123)
		assert(f, err)
		assert(f() == "num-ok")
	`
	runLuaSourceWithProvider(t, source, "test", provider)
}

func TestLoadfile_BadFilenameType(t *testing.T) {
	provider := newMockProvider(map[string]string{})
	source := `
		local ok, err = pcall(loadfile, {})
		assert(not ok, "expected failure")
		assert(type(err) == "string")
		assert(err:find("bad argument #1 to 'loadfile' %(string expected, got table%)") ~= nil,
			"unexpected error: " .. tostring(err))
	`
	runLuaSourceWithProvider(t, source, "test", provider)
}

func TestDofile_NumberFilenameCoercion(t *testing.T) {
	provider := newMockProvider(map[string]string{
		"123": `return "num-ok"`,
	})
	source := `
		local got = dofile(123)
		assert(got == "num-ok", "expected num-ok, got " .. tostring(got))
	`
	runLuaSourceWithProvider(t, source, "test", provider)
}

func TestDofile_BadFilenameType(t *testing.T) {
	provider := newMockProvider(map[string]string{})
	source := `
		local ok, err = pcall(dofile, {})
		assert(not ok, "expected failure")
		assert(type(err) == "string")
		assert(err:find("bad argument #1 to 'dofile' %(string expected, got table%)") ~= nil,
			"unexpected error: " .. tostring(err))
	`
	runLuaSourceWithProvider(t, source, "test", provider)
}

func TestLoadfile_NotPermitted(t *testing.T) {
	provider := &MockProvider{
		files: map[string]string{"x.lua": "return 1"},
		caps:  vm.LuaLoaderCaps{AllowDofile: true, AllowLoadfile: false},
	}
	source := `
		assert(loadfile == nil, "expected loadfile to be nil when not permitted")
	`
	runLuaSourceWithProvider(t, source, "test", provider)
}

type contextRecordingProvider struct {
	files map[string]string
	caps  vm.LuaLoaderCaps
	calls []providerCall
}

type providerCall struct {
	name   string
	caller vm.LuaCallerContext
}

func (p *contextRecordingProvider) LoadChunk(name string, caller *vm.LuaCallerContext) ([]byte, string, error) {
	if caller != nil {
		p.calls = append(p.calls, providerCall{name: name, caller: *caller})
	} else {
		p.calls = append(p.calls, providerCall{name: name})
	}
	src, ok := p.files[name]
	if !ok {
		return nil, "", fmt.Errorf("cannot open '%s': no such file", name)
	}
	return []byte(src), "@" + name, nil
}

func (p *contextRecordingProvider) Capabilities() vm.LuaLoaderCaps {
	return p.caps
}

func TestDofile_CallerContext(t *testing.T) {
	provider := &contextRecordingProvider{
		files: map[string]string{
			"outer.lua": `return dofile("inner.lua")`,
			"inner.lua": `return "ok"`,
		},
		caps: vm.LuaLoaderCaps{AllowDofile: true, AllowLoadfile: true},
	}

	block, err := parser.Parse("test_context", `assert(dofile("outer.lua") == "ok")`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("test_context", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	v := vm.New()
	v.SetVMID("vm-context")
	v.SetCodeProvider(provider)
	stdlib.Open(v)
	if _, err := v.Run(proto); err != nil {
		t.Fatalf("runtime error: %v", err)
	}

	if len(provider.calls) != 2 {
		t.Fatalf("expected 2 provider calls, got %d", len(provider.calls))
	}
	if provider.calls[0].name != "outer.lua" {
		t.Fatalf("first call name = %q, expected outer.lua", provider.calls[0].name)
	}
	if provider.calls[1].name != "inner.lua" {
		t.Fatalf("second call name = %q, expected inner.lua", provider.calls[1].name)
	}
	if provider.calls[0].caller.VMID != "vm-context" || provider.calls[1].caller.VMID != "vm-context" {
		t.Fatalf("expected VMID vm-context in both calls, got %+v and %+v", provider.calls[0].caller, provider.calls[1].caller)
	}
	if provider.calls[0].caller.ScriptName != "" {
		t.Fatalf("first call ScriptName = %q, expected empty top-level script name", provider.calls[0].caller.ScriptName)
	}
	if provider.calls[1].caller.ScriptName != "@outer.lua" {
		t.Fatalf("second call ScriptName = %q, expected @outer.lua", provider.calls[1].caller.ScriptName)
	}
	if provider.calls[1].caller.CallDepth <= provider.calls[0].caller.CallDepth {
		t.Fatalf("expected nested call depth to increase, got %d then %d", provider.calls[0].caller.CallDepth, provider.calls[1].caller.CallDepth)
	}
}

func TestLoadfile_CompileErrorIncludesChunkName(t *testing.T) {
	provider := newMockProvider(map[string]string{
		"broken.lua": `return 1 +`,
	})
	source := `
		local f, err = loadfile("broken.lua")
		assert(f == nil, "expected nil on compile error")
		assert(type(err) == "string", "expected string compile error")
		assert(err:find("broken.lua", 1, true) ~= nil, "expected chunk name in error: " .. tostring(err))
	`
	runLuaSourceWithProvider(t, source, "test", provider)
}

func TestLoadfile_MissingChunkProviderError(t *testing.T) {
	provider := newMockProvider(map[string]string{})
	source := `
		local f, err = loadfile("missing.lua")
		assert(f == nil, "expected nil")
		assert(type(err) == "string", "expected string error")
		assert(err:find("cannot open", 1, true) ~= nil, "unexpected error: " .. tostring(err))
	`
	runLuaSourceWithProvider(t, source, "test", provider)
}

func TestLoadfile_EmptyFilenamePassesToProvider(t *testing.T) {
	provider := newMockProvider(map[string]string{"": `return "empty"`})
	source := `
		local f, err = loadfile()
		assert(f, err)
		assert(f() == "empty", "expected provider-backed empty-name chunk")
	`
	runLuaSourceWithProvider(t, source, "test", provider)
}

func TestDofile_EmptyFilenamePassesToProvider(t *testing.T) {
	provider := newMockProvider(map[string]string{"": `return "empty"`})
	source := `
		local r = dofile()
		assert(r == "empty", "expected provider-backed empty-name chunk")
	`
	runLuaSourceWithProvider(t, source, "test", provider)
}

func TestDofile_CompileErrorMessageIncludesChunkName(t *testing.T) {
	provider := newMockProvider(map[string]string{
		"bad.lua": `return 1 +`,
	})
	source := `
		local ok, err = pcall(dofile, "bad.lua")
		assert(not ok, "expected failure")
		assert(type(err) == "string", "expected string error")
		assert(err:find("bad.lua", 1, true) ~= nil, "expected chunk name in error: " .. tostring(err))
	`
	runLuaSourceWithProvider(t, source, "test", provider)
}

func TestLoadfile_MissingChunkErrorShape(t *testing.T) {
	provider := newMockProvider(map[string]string{})
	source := `
		local f, err = loadfile(999)
		assert(f == nil, "expected nil")
		assert(type(err) == "string", "expected string error")
		assert(err:find("999", 1, true) ~= nil, "expected coerced filename in error")
	`
	runLuaSourceWithProvider(t, source, "test", provider)
}

func TestDofile_MissingChunkErrorShape(t *testing.T) {
	provider := newMockProvider(map[string]string{})
	source := `
		local ok, err = pcall(dofile, 999)
		assert(not ok, "expected failure")
		assert(type(err) == "string", "expected string error")
		assert(err:find("999", 1, true) ~= nil, "expected coerced filename in error")
	`
	runLuaSourceWithProvider(t, source, "test", provider)
}

func TestLoadfile_NoImplicitFilesystemAssumption(t *testing.T) {
	provider := newMockProvider(map[string]string{})
	source := `
		local f, err = loadfile("not-on-disk.lua")
		assert(f == nil, "expected nil")
		assert(type(err) == "string", "expected string error")
		assert(err:find("cannot open", 1, true) ~= nil, "expected provider error")
	`
	runLuaSourceWithProvider(t, source, "test", provider)
}
