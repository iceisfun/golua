package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

func runLuaWithIo(t *testing.T, source, name string, provider vm.LuaIoProvider) {
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
	v.SetIoProvider(provider)
	stdlib.Open(v)

	_, err = v.Run(proto)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

func TestIo_OpenReadAll(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "hello.txt"), []byte("Hello, World!"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	provider := vm.NewJailedIoProvider(tmpDir)
	source := `
		local f = io.open("hello.txt", "r")
		assert(f, "failed to open file")
		local content = f:read("*a")
		assert(content == "Hello, World!", "expected 'Hello, World!', got '" .. tostring(content) .. "'")
		f:close()
	`
	runLuaWithIo(t, source, "test_io_read_all", provider)
}

func TestIo_ReadLine(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "lines.txt"), []byte("line1\nline2\nline3\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	provider := vm.NewJailedIoProvider(tmpDir)
	source := `
		local f = io.open("lines.txt", "r")
		assert(f, "failed to open file")

		local l1 = f:read("*l")
		assert(l1 == "line1", "expected 'line1', got '" .. tostring(l1) .. "'")

		local l2 = f:read("*l")
		assert(l2 == "line2", "expected 'line2', got '" .. tostring(l2) .. "'")

		local l3 = f:read("*l")
		assert(l3 == "line3", "expected 'line3', got '" .. tostring(l3) .. "'")

		f:close()
	`
	runLuaWithIo(t, source, "test_io_read_line", provider)
}

func TestIo_ReadLineWithNewline(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "lines.txt"), []byte("line1\nline2\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	provider := vm.NewJailedIoProvider(tmpDir)
	source := `
		local f = io.open("lines.txt", "r")
		assert(f, "failed to open file")

		local l1 = f:read("*L")
		assert(l1 == "line1\n", "expected 'line1\\n', got '" .. tostring(l1) .. "'")

		local l2 = f:read("*L")
		assert(l2 == "line2\n", "expected 'line2\\n', got '" .. tostring(l2) .. "'")

		f:close()
	`
	runLuaWithIo(t, source, "test_io_read_line_newline", provider)
}

func TestIo_ReadBytes(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "data.txt"), []byte("ABCDEFGHIJ"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	provider := vm.NewJailedIoProvider(tmpDir)
	source := `
		local f = io.open("data.txt", "r")
		assert(f, "failed to open file")

		local chunk = f:read(5)
		assert(chunk == "ABCDE", "expected 'ABCDE', got '" .. tostring(chunk) .. "'")

		local chunk2 = f:read(5)
		assert(chunk2 == "FGHIJ", "expected 'FGHIJ', got '" .. tostring(chunk2) .. "'")

		f:close()
	`
	runLuaWithIo(t, source, "test_io_read_bytes", provider)
}

func TestIo_Lines(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "poem.txt"), []byte("roses\nviolets\nsugar\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	provider := vm.NewJailedIoProvider(tmpDir)
	source := `
		local lines = {}
		for line in io.lines("poem.txt") do
			lines[#lines + 1] = line
		end
		assert(#lines == 3, "expected 3 lines, got " .. tostring(#lines))
		assert(lines[1] == "roses", "expected 'roses', got '" .. tostring(lines[1]) .. "'")
		assert(lines[2] == "violets", "expected 'violets', got '" .. tostring(lines[2]) .. "'")
		assert(lines[3] == "sugar", "expected 'sugar', got '" .. tostring(lines[3]) .. "'")
	`
	runLuaWithIo(t, source, "test_io_lines", provider)
}

func TestIo_Type(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("data"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	provider := vm.NewJailedIoProvider(tmpDir)
	source := `
		local f = io.open("test.txt", "r")
		assert(io.type(f) == "file", "expected 'file', got '" .. tostring(io.type(f)) .. "'")

		f:close()
		assert(io.type(f) == "closed file", "expected 'closed file', got '" .. tostring(io.type(f)) .. "'")

		assert(io.type("hello") == nil, "expected nil for non-file")
		assert(io.type(42) == nil, "expected nil for number")
	`
	runLuaWithIo(t, source, "test_io_type", provider)
}

func TestIo_Close(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("data"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	provider := vm.NewJailedIoProvider(tmpDir)
	source := `
		local f = io.open("test.txt", "r")
		assert(f, "failed to open file")
		io.close(f)
		assert(io.type(f) == "closed file", "expected 'closed file' after io.close")
	`
	runLuaWithIo(t, source, "test_io_close", provider)
}

func TestIo_DoubleCloseError(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("data"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	provider := vm.NewJailedIoProvider(tmpDir)
	source := `
		local f = io.open("test.txt", "r")
		f:close()
		local ok, err = pcall(function() f:close() end)
		assert(not ok, "expected error on double close")
	`
	runLuaWithIo(t, source, "test_io_double_close", provider)
}

func TestIo_NoProvider(t *testing.T) {
	source := `
		assert(io == nil, "expected io to be nil without provider")
	`
	runLuaSource(t, source, "test_io_no_provider")
}

func TestIo_WriteModeRejected(t *testing.T) {
	tmpDir := t.TempDir()
	provider := vm.NewJailedIoProvider(tmpDir)
	source := `
		local f, err = io.open("output.txt", "w")
		assert(f == nil, "expected nil for write mode")
		assert(err, "expected error message for write mode")
	`
	runLuaWithIo(t, source, "test_io_write_rejected", provider)
}

func TestIo_PathTraversalRejected(t *testing.T) {
	tmpDir := t.TempDir()
	provider := vm.NewJailedIoProvider(tmpDir)
	source := `
		local f, err = io.open("../../../etc/passwd", "r")
		assert(f == nil, "expected nil for path traversal")
		assert(err, "expected error message for path traversal")
	`
	runLuaWithIo(t, source, "test_io_path_traversal", provider)
}

func TestIo_FileLines(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "data.txt"), []byte("alpha\nbeta\ngamma\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	provider := vm.NewJailedIoProvider(tmpDir)
	source := `
		local f = io.open("data.txt", "r")
		local lines = {}
		for line in f:lines() do
			lines[#lines + 1] = line
		end
		f:close()
		assert(#lines == 3, "expected 3 lines, got " .. tostring(#lines))
		assert(lines[1] == "alpha")
		assert(lines[2] == "beta")
		assert(lines[3] == "gamma")
	`
	runLuaWithIo(t, source, "test_io_file_lines", provider)
}
