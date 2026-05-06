package stdlib

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/vm"
)

func compileTestLua(source string) (*compiler.Proto, error) {
	ast, err := parser.Parse("=test", source)
	if err != nil {
		return nil, err
	}
	return compiler.Compile("=test", ast)
}

// runLua compiles and executes a Lua source string with full stdlib + FullIoProvider + OsProvider.
func runLua(t *testing.T, source string) error {
	t.Helper()
	proto, err := compileTestLua(source)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	v := vm.New()
	tmpDir := t.TempDir()
	v.SetIoProvider(vm.NewFullIoProvider(tmpDir))
	v.SetOsProvider(vm.NewDefaultOsProvider())
	Open(v)
	_, err = v.Run(proto)
	return err
}

func runLuaExpectError(t *testing.T, source, substr string) {
	t.Helper()
	err := runLua(t, source)
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", substr)
	}
	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("expected error containing %q, got: %s", substr, err.Error())
	}
}

func runLuaExpectOK(t *testing.T, source string) {
	t.Helper()
	err := runLua(t, source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0666)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}

func runLuaWithDir(t *testing.T, dir string, source string) {
	t.Helper()
	proto, err := compileTestLua(source)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	v := vm.New()
	v.SetIoProvider(vm.NewFullIoProvider(dir))
	v.SetOsProvider(vm.NewDefaultOsProvider())
	Open(v)
	_, runErr := v.Run(proto)
	if runErr != nil {
		t.Fatalf("runtime error: %v", runErr)
	}
}

func runLuaWithDirExpectError(t *testing.T, dir string, source, substr string) {
	t.Helper()
	proto, err := compileTestLua(source)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	v := vm.New()
	v.SetIoProvider(vm.NewFullIoProvider(dir))
	v.SetOsProvider(vm.NewDefaultOsProvider())
	Open(v)
	_, runErr := v.Run(proto)
	if runErr == nil {
		t.Fatalf("expected error containing %q, got nil", substr)
	}
	if !strings.Contains(runErr.Error(), substr) {
		t.Fatalf("expected error containing %q, got: %s", substr, runErr.Error())
	}
}

// --- Bug 1: file:read("n") always returns float, never integer ---
func TestFileReadN_IntegerResult(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "num.txt", "42")
	runLuaWithDir(t, dir, `local f = io.open("num.txt", "r")
local v = f:read("n")
f:close()
assert(v == 42, "value should be 42, got " .. tostring(v))
assert(math.type(v) == "integer", "expected integer, got " .. math.type(v))`)
}

func TestFileReadN_FloatResult(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "num.txt", "42.5")
	runLuaWithDir(t, dir, `local f = io.open("num.txt", "r")
local v = f:read("n")
f:close()
assert(v == 42.5, "value should be 42.5, got " .. tostring(v))
assert(math.type(v) == "float", "expected float, got " .. math.type(v))`)
}

// --- Bug 2: file:read("n") does not parse hex numbers ---
func TestFileReadN_HexNumber(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "hex.txt", "0xFF")
	runLuaWithDir(t, dir, `local f = io.open("hex.txt", "r")
local v = f:read("n")
f:close()
assert(v == 255, "value should be 255, got " .. tostring(v))
assert(math.type(v) == "integer", "expected integer, got " .. math.type(v))`)
}

// --- Bug 3: file:read("n") failure leaves file position wrong ---
func TestFileReadN_FailedReadPosition(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "mixed.txt", "abc 123")
	runLuaWithDir(t, dir, `local f = io.open("mixed.txt", "r")
local v = f:read("n")
assert(v == nil, "reading 'abc' as number should fail, got " .. tostring(v))
-- Position should be unchanged (at 0), so reading a line should get the full content
local line = f:read("l")
assert(line == "abc 123", "expected 'abc 123', got '" .. tostring(line) .. "'")
f:close()`)
}

// --- Bug 4: io.write does not return io.stdout ---
func TestIoWriteReturnsStdout(t *testing.T) {
	runLuaExpectOK(t, `local result = io.write("")
assert(result == io.stdout, "io.write should return io.stdout")`)
}

// --- Bug 5: io.input()/io.output() identity mismatch ---
func TestIoInputOutputIdentity(t *testing.T) {
	runLuaExpectOK(t, `assert(io.input() == io.stdin, "io.input() should == io.stdin")
assert(io.output() == io.stdout, "io.output() should == io.stdout")`)
}

// --- Bug 6: io.open with invalid mode should hard-error ---
func TestIoOpenInvalidMode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "test.txt", "hello")
	runLuaWithDirExpectError(t, dir, `io.open("test.txt", "z")`, "invalid mode")
}

// --- Bug 7: io.type() with no arguments doesn't error ---
func TestIoTypeNoArgs(t *testing.T) {
	runLuaExpectError(t, `io.type()`, "bad argument #1")
}

// --- Bug 8: file:setvbuf return type ---
func TestFileSetvbufReturnsTrue(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "test.txt", "hello")
	runLuaWithDir(t, dir, `local f = io.open("test.txt", "r")
local result = f:setvbuf("no")
assert(result == true, "setvbuf should return true (boolean), got " .. type(result))
assert(type(result) == "boolean", "setvbuf should return boolean, got " .. type(result))
f:close()`)
}

// --- Bug 9: file:seek with invalid whence doesn't error ---
func TestFileSeekInvalidWhence(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "test.txt", "hello")
	runLuaWithDirExpectError(t, dir, `local f = io.open("test.txt", "r")
f:seek("invalid")
f:close()`, "invalid option 'invalid'")
}

// --- Bug 10: file:read with invalid format doesn't error ---
func TestFileReadInvalidFormat(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "test.txt", "hello")
	runLuaWithDirExpectError(t, dir, `local f = io.open("test.txt", "r")
f:read("z")
f:close()`, "invalid format")
}

// --- Bug 11: os.remove error format ---
func TestOsRemoveErrorFormat(t *testing.T) {
	dir := t.TempDir()
	runLuaWithDir(t, dir, `local ok, msg, errno = os.remove("/nonexistent_path_xyz_12345")
assert(ok == nil, "expected nil")
assert(type(msg) == "string", "expected string error message")
assert(type(errno) == "number", "expected numeric errno")
-- Message should NOT have "remove" prefix (Go os.Remove format)
assert(not string.find(msg, "^remove "), "error message should not start with 'remove': " .. msg)
-- Should be: "/path: Error description"
assert(string.find(msg, "^/"), "error message should start with path: " .. msg)`)
}

// --- Bug 12: os.rename error format ---
func TestOsRenameErrorFormat(t *testing.T) {
	dir := t.TempDir()
	runLuaWithDir(t, dir, `local ok, msg, errno = os.rename("/nonexistent_src_xyz", "/nonexistent_dst_xyz")
assert(ok == nil, "expected nil")
assert(type(msg) == "string", "expected string error message")
assert(type(errno) == "number", "expected numeric errno")
-- Message should NOT have "rename" prefix
assert(not string.find(msg, "^rename "), "error message should not start with 'rename': " .. msg)`)
}

// --- Bug 13: io.open error message format ---
func TestIoOpenErrorMessageFormat(t *testing.T) {
	dir := t.TempDir()
	runLuaWithDir(t, dir, `local ok, msg = io.open("/tmp/nonexistent_xyz_12345", "r")
assert(ok == nil, "expected nil")
-- Should NOT double the path (Go's os.OpenFile returns "open /path: ...")
-- Count occurrences of the path
local _, count = string.gsub(msg, "/tmp/nonexistent_xyz_12345", "X")
assert(count == 1, "path should appear exactly once in error: " .. msg)`)
}

// --- Bug 14: io.lines error message format ---
func TestIoLinesErrorMessageFormat(t *testing.T) {
	runLuaExpectError(t, `io.lines("/nonexistent_file_xyz_12345")`,
		"cannot open file '/nonexistent_file_xyz_12345'")
}

// --- Bug 15: os.date with trailing % error ---
func TestOsDateTrailingPercent(t *testing.T) {
	// Lua 5.4 says: invalid conversion specifier '%'
	runLuaExpectError(t, `os.date("%")`, `'%'`)
}

// --- Bug 16: io.tmpfile ---
func TestIoTmpfile(t *testing.T) {
	dir := t.TempDir()
	runLuaWithDir(t, dir, `local f = io.tmpfile()
assert(f ~= nil, "io.tmpfile should return a file handle")
assert(io.type(f) == "file", "io.tmpfile result should be a file")
f:write("hello")
f:seek("set", 0)
local data = f:read("a")
assert(data == "hello", "expected 'hello', got '" .. tostring(data) .. "'")
f:close()`)
}

// --- Bug 17: file:setvbuf with invalid mode doesn't hard-error ---
func TestFileSetvbufInvalidMode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "test.txt", "hello")
	runLuaWithDirExpectError(t, dir, `local f = io.open("test.txt", "r")
f:setvbuf("invalid")
f:close()`, "invalid option 'invalid'")
}

// --- Bug 18: io.close() no-arg should return nil, "cannot close standard file" ---
func TestIoCloseNoArg(t *testing.T) {
	runLuaExpectOK(t, `
local ok, msg = io.close()
assert(ok == nil, "expected nil, got " .. tostring(ok))
assert(msg == "cannot close standard file", "expected 'cannot close standard file', got " .. tostring(msg))
`)
}

func TestIoCloseNoArgClosesCurrentDefaultOutput(t *testing.T) {
	dir := t.TempDir()
	runLuaWithDir(t, dir, `
local f = assert(io.open("out.txt", "w"))
assert(io.output(f) == f)
assert(io.close() == true)
assert(io.output() == f)
assert(io.type(f) == "closed file")
`)
}

func TestIoReadNegativeCount(t *testing.T) {
	runLuaExpectError(t, `io.read(-1)`, "resulting string too large")
}

func TestFileReadNegativeCount(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "test.txt", "hello")
	runLuaWithDirExpectError(t, dir, `local f = assert(io.open("test.txt", "r")); f:read(-1)`, "resulting string too large")
}

func TestFileReadNonIntegralCount(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "test.txt", "hello")
	runLuaWithDirExpectError(t, dir, `local f = assert(io.open("test.txt", "r")); f:read(-1.5)`, "number has no integer representation")
}

// --- Bug 19: file:seek invalid whence arg number ---
// Method syntax: name resolved from bytecode, arg decremented for self.
func TestFileSeekInvalidWhenceArgNum(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "test.txt", "hello")
	runLuaWithDirExpectError(t, dir, `local f = io.open("test.txt", "r")
f:seek("invalid")
f:close()`, "bad argument #1")
}

// --- Bug 20: file:setvbuf invalid mode arg number ---
// Method syntax: name resolved from bytecode, arg decremented for self.
func TestFileSetvbufInvalidModeArgNum(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "test.txt", "hello")
	runLuaWithDirExpectError(t, dir, `local f = io.open("test.txt", "r")
f:setvbuf("invalid")
f:close()`, "bad argument #1")
}

// --- Bug 21: io.lines(filename) should return 4 values ---
func TestIoLinesFileReturns4Values(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "test.txt", "hello\nworld\n")
	runLuaWithDir(t, dir, `
local a, b, c, d = io.lines("test.txt")
assert(type(a) == "function", "1st return should be function, got " .. type(a))
assert(b == nil, "2nd return should be nil")
assert(c == nil, "3rd return should be nil")
assert(io.type(d) == "file", "4th return should be file handle, got " .. tostring(io.type(d)))
`)
}

// --- Bug 22: io.lines auto-close via generic for ---
func TestIoLinesAutoClose(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "test.txt", "hello\nworld\n")
	runLuaWithDir(t, dir, `
local lines = {}
for line in io.lines("test.txt") do
    lines[#lines+1] = line
end
assert(#lines == 2, "expected 2 lines, got " .. #lines)
assert(lines[1] == "hello", "expected 'hello', got " .. lines[1])
assert(lines[2] == "world", "expected 'world', got " .. lines[2])
`)
}

// --- Bug 23: io.input error format should use "cannot open file 'path' (error)" ---
func TestIoInputErrorFormat(t *testing.T) {
	dir := t.TempDir()
	runLuaWithDirExpectError(t, dir,
		`io.input("nonexistent_xyz_12345")`,
		"cannot open file 'nonexistent_xyz_12345'")
}

// --- Provider interface tests ---

func TestFullIoProvider_TmpFile(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	p := vm.NewFullIoProvider(dir)
	f, err := p.TmpFile(ctx)
	if err != nil {
		t.Fatalf("TmpFile failed: %v", err)
	}
	defer f.Close(ctx)

	err = f.Write(ctx, "test data")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	_, err = f.Seek(ctx, "set", 0)
	if err != nil {
		t.Fatalf("Seek failed: %v", err)
	}
	data, err := f.Read(ctx, "a")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if data != "test data" {
		t.Fatalf("expected 'test data', got %q", data)
	}
}

func TestFullIoProvider_Open(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	p := vm.NewFullIoProvider(dir)

	f, err := p.Open(ctx, "test.txt", "w")
	if err != nil {
		t.Fatalf("Open w failed: %v", err)
	}
	f.Write(ctx, "hello world")
	f.Close(ctx)

	f, err = p.Open(ctx, "test.txt", "r")
	if err != nil {
		t.Fatalf("Open r failed: %v", err)
	}
	data, err := f.Read(ctx, "a")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	f.Close(ctx)

	if data != "hello world" {
		t.Fatalf("expected 'hello world', got %q", data)
	}
}

func TestFullIoProvider_InvalidMode(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	p := vm.NewFullIoProvider(dir)
	_, err := p.Open(ctx, "test.txt", "z")
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestFullIoProvider_SeekErrors(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	p := vm.NewFullIoProvider(dir)
	f, err := p.Open(ctx, "test.txt", "w")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer f.Close(ctx)

	_, err = f.Seek(ctx, "invalid", 0)
	if err == nil {
		t.Fatal("expected error for invalid whence")
	}
	if !strings.Contains(err.Error(), "invalid option") {
		t.Fatalf("expected 'invalid option' error, got: %v", err)
	}
}

func TestFullIoProvider_SetVBuf(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	p := vm.NewFullIoProvider(dir)
	f, err := p.Open(ctx, "test.txt", "w")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer f.Close(ctx)

	for _, mode := range []string{"no", "full", "line"} {
		err = f.SetVBuf(ctx, mode, 0)
		if err != nil {
			t.Fatalf("SetVBuf(%q) failed: %v", mode, err)
		}
	}

	err = f.SetVBuf(ctx, "invalid", 0)
	if err == nil {
		t.Fatal("expected error for invalid setvbuf mode")
	}
	if !strings.Contains(err.Error(), "invalid option") {
		t.Fatalf("expected 'invalid option' error, got: %v", err)
	}
}

func TestFullIoProvider_ReadNumber(t *testing.T) {
	dir := t.TempDir()
	p := vm.NewFullIoProvider(dir)

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{"integer", "42", "42"},
		{"negative", "-7", "-7"},
		{"float", "3.14", "3.14"},
		{"hex", "0xFF", "0xFF"},
		{"hex_upper", "0XAB", "0XAB"},
		{"scientific", "1e10", "1e10"},
		{"leading_space", "  42", "42"},
	}

	ctx := context.Background()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			os.WriteFile(filepath.Join(dir, "num.txt"), []byte(tc.content), 0666)
			f, err := p.Open(ctx, "num.txt", "r")
			if err != nil {
				t.Fatalf("Open failed: %v", err)
			}
			data, err := f.Read(ctx, "n")
			if err != nil {
				t.Fatalf("Read('n') failed: %v", err)
			}
			f.Close(ctx)
			if data != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, data)
			}
		})
	}
}

func TestFullIoProvider_ReadNumberFail(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	p := vm.NewFullIoProvider(dir)
	os.WriteFile(filepath.Join(dir, "abc.txt"), []byte("abc 123"), 0666)
	f, err := p.Open(ctx, "abc.txt", "r")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer f.Close(ctx)

	_, err = f.Read(ctx, "n")
	if err == nil {
		t.Fatal("expected error reading non-number")
	}
	// After failed read, position should be unchanged
	data, err := f.Read(ctx, "l")
	if err != nil {
		t.Fatalf("Read('l') after failed Read('n') failed: %v", err)
	}
	if data != "abc 123" {
		t.Fatalf("expected 'abc 123' after failed read, got %q", data)
	}
}

func TestFullIoProvider_Remove(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	p := vm.NewFullIoProvider(dir)

	f, _ := p.Open(ctx, "removeme.txt", "w")
	f.Close(ctx)

	err := p.Remove(ctx, "removeme.txt")
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	err = p.Remove(ctx, "nonexistent.txt")
	if err == nil {
		t.Fatal("expected error for removing nonexistent file")
	}
}

func TestFullIoProvider_Rename(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	p := vm.NewFullIoProvider(dir)

	f, _ := p.Open(ctx, "old.txt", "w")
	f.Write(ctx, "content")
	f.Close(ctx)

	err := p.Rename(ctx, "old.txt", "new.txt")
	if err != nil {
		t.Fatalf("Rename failed: %v", err)
	}

	_, err = p.Open(ctx, "old.txt", "r")
	if err == nil {
		t.Fatal("expected error opening old name after rename")
	}

	f, err = p.Open(ctx, "new.txt", "r")
	if err != nil {
		t.Fatalf("Open new name failed: %v", err)
	}
	data, _ := f.Read(ctx, "a")
	f.Close(ctx)
	if data != "content" {
		t.Fatalf("expected 'content', got %q", data)
	}
}

func TestFullIoProvider_StdFiles(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	p := vm.NewFullIoProvider(dir)

	stdin := p.Stdin(ctx)
	if stdin == nil {
		t.Fatal("Stdin should not be nil")
	}
	if !stdin.IsStd(ctx) {
		t.Fatal("Stdin should be std")
	}

	stdout := p.Stdout(ctx)
	if stdout == nil {
		t.Fatal("Stdout should not be nil")
	}
	if !stdout.IsStd(ctx) {
		t.Fatal("Stdout should be std")
	}

	stderr := p.Stderr(ctx)
	if stderr == nil {
		t.Fatal("Stderr should not be nil")
	}
	if !stderr.IsStd(ctx) {
		t.Fatal("Stderr should be std")
	}
}

func TestFullIoProvider_Capabilities(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	p := vm.NewFullIoProvider(dir)
	caps := p.Capabilities(ctx)
	if !caps.AllowRead {
		t.Fatal("AllowRead should be true")
	}
	if !caps.AllowWrite {
		t.Fatal("AllowWrite should be true")
	}
}

func TestJailedIoProvider_TmpFile(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	p := vm.NewJailedIoProvider(dir)
	_, err := p.TmpFile(ctx)
	if err == nil {
		t.Fatal("expected error from jailed TmpFile")
	}
}

func TestJailedIoProvider_ReadOnly(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0666)
	p := vm.NewJailedIoProvider(dir)

	// Read should work
	f, err := p.Open(ctx, "test.txt", "r")
	if err != nil {
		t.Fatalf("Open r failed: %v", err)
	}
	data, err := f.Read(ctx, "a")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	f.Close(ctx)
	if data != "hello" {
		t.Fatalf("expected 'hello', got %q", data)
	}

	// Write should fail
	_, err = p.Open(ctx, "test.txt", "w")
	if err == nil {
		t.Fatal("expected error for write mode in jailed provider")
	}
}
