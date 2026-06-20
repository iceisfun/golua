package golua_test

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

func TestIo_LinesWithFormats(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "data.txt"), []byte("hello\nworld\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	provider := vm.NewJailedIoProvider(tmpDir)
	source := `
		local lines = {}
		for line in io.lines("data.txt", "L") do
			lines[#lines + 1] = line
		end
		assert(#lines == 2, "expected 2 lines, got " .. tostring(#lines))
		assert(lines[1] == "hello\n", "expected 'hello\\n', got '" .. tostring(lines[1]) .. "'")
		assert(lines[2] == "world\n", "expected 'world\\n', got '" .. tostring(lines[2]) .. "'")
	`
	runLuaWithIo(t, source, "test_io_lines_L_format", provider)
}

func TestIo_LinesWithByteFormat(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "data.txt"), []byte("ABCDEFGH"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	provider := vm.NewJailedIoProvider(tmpDir)
	source := `
		local chunks = {}
		for chunk in io.lines("data.txt", 3) do
			chunks[#chunks + 1] = chunk
		end
		assert(#chunks == 3, "expected 3 chunks, got " .. tostring(#chunks))
		assert(chunks[1] == "ABC", "expected 'ABC', got '" .. tostring(chunks[1]) .. "'")
		assert(chunks[2] == "DEF", "expected 'DEF', got '" .. tostring(chunks[2]) .. "'")
		assert(chunks[3] == "GH", "expected 'GH', got '" .. tostring(chunks[3]) .. "'")
	`
	runLuaWithIo(t, source, "test_io_lines_byte_format", provider)
}

func TestIo_LinesWithMultipleFormats(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "data.txt"), []byte("hello\nworld\nfoo\nbar\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	provider := vm.NewJailedIoProvider(tmpDir)
	source := `
		local pairs = {}
		for a, b in io.lines("data.txt", "l", "l") do
			pairs[#pairs + 1] = a .. "+" .. b
		end
		assert(#pairs == 2, "expected 2 pairs, got " .. tostring(#pairs))
		assert(pairs[1] == "hello+world", "expected 'hello+world', got '" .. tostring(pairs[1]) .. "'")
		assert(pairs[2] == "foo+bar", "expected 'foo+bar', got '" .. tostring(pairs[2]) .. "'")
	`
	runLuaWithIo(t, source, "test_io_lines_multi_format", provider)
}

func TestIo_LinesWithNumberFormat(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "data.txt"), []byte("42\n3.14\n100\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	provider := vm.NewJailedIoProvider(tmpDir)
	source := `
		local nums = {}
		for n in io.lines("data.txt", "n") do
			nums[#nums + 1] = n
		end
		assert(#nums == 3, "expected 3 numbers, got " .. tostring(#nums))
		assert(nums[1] == 42, "expected 42, got " .. tostring(nums[1]))
		assert(nums[2] == 3.14, "expected 3.14, got " .. tostring(nums[2]))
		assert(nums[3] == 100, "expected 100, got " .. tostring(nums[3]))
	`
	runLuaWithIo(t, source, "test_io_lines_number_format", provider)
}

func TestIo_LinesAutoClose(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "data.txt"), []byte("one\ntwo\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	provider := vm.NewJailedIoProvider(tmpDir)
	source := `
		-- io.lines should auto-close the file after iteration finishes
		local iter = io.lines("data.txt", "l")
		local a = iter()
		assert(a == "one", "expected 'one', got '" .. tostring(a) .. "'")
		local b = iter()
		assert(b == "two", "expected 'two', got '" .. tostring(b) .. "'")
		-- This call hits EOF, should auto-close
		local c = iter()
		assert(c == nil, "expected nil at EOF, got '" .. tostring(c) .. "'")
		-- Calling again after close should error
		local ok, err = pcall(iter)
		assert(not ok, "expected error calling iterator after auto-close")
	`
	runLuaWithIo(t, source, "test_io_lines_auto_close", provider)
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

func TestIo_FileLinesWithFormat(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "data.txt"), []byte("alpha\nbeta\ngamma\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	provider := vm.NewJailedIoProvider(tmpDir)
	source := `
		local f = io.open("data.txt", "r")
		local lines = {}
		for line in f:lines("L") do
			lines[#lines + 1] = line
		end
		f:close()
		assert(#lines == 3, "expected 3 lines, got " .. tostring(#lines))
		assert(lines[1] == "alpha\n", "expected 'alpha\\n'")
		assert(lines[2] == "beta\n", "expected 'beta\\n'")
		assert(lines[3] == "gamma\n", "expected 'gamma\\n'")
	`
	runLuaWithIo(t, source, "test_io_file_lines_format", provider)
}

func TestIo_FileLinesWithByteFormat(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "data.txt"), []byte("ABCDEF"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	provider := vm.NewJailedIoProvider(tmpDir)
	source := `
		local f = io.open("data.txt", "r")
		local chunks = {}
		for chunk in f:lines(2) do
			chunks[#chunks + 1] = chunk
		end
		f:close()
		assert(#chunks == 3, "expected 3 chunks, got " .. tostring(#chunks))
		assert(chunks[1] == "AB", "expected 'AB', got '" .. tostring(chunks[1]) .. "'")
		assert(chunks[2] == "CD", "expected 'CD', got '" .. tostring(chunks[2]) .. "'")
		assert(chunks[3] == "EF", "expected 'EF', got '" .. tostring(chunks[3]) .. "'")
	`
	runLuaWithIo(t, source, "test_io_file_lines_byte", provider)
}

func TestIo_FileLinesMultipleFormats(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "data.txt"), []byte("hello\nworld\nfoo\nbar\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	provider := vm.NewJailedIoProvider(tmpDir)
	source := `
		local f = io.open("data.txt", "r")
		local pairs = {}
		for a, b in f:lines("l", "l") do
			pairs[#pairs + 1] = a .. "+" .. b
		end
		f:close()
		assert(#pairs == 2, "expected 2 pairs, got " .. tostring(#pairs))
		assert(pairs[1] == "hello+world", "expected 'hello+world', got '" .. tostring(pairs[1]) .. "'")
		assert(pairs[2] == "foo+bar", "expected 'foo+bar', got '" .. tostring(pairs[2]) .. "'")
	`
	runLuaWithIo(t, source, "test_io_file_lines_multi", provider)
}

func TestIo_FileLinesDoesNotAutoClose(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "data.txt"), []byte("one\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	provider := vm.NewJailedIoProvider(tmpDir)
	source := `
		local f = io.open("data.txt", "r")
		for line in f:lines() do end
		-- f:lines() should NOT auto-close the file; caller manages lifetime
		assert(io.type(f) == "file", "expected file still open after f:lines() exhausted, got " .. tostring(io.type(f)))
		f:close()
	`
	runLuaWithIo(t, source, "test_io_file_lines_no_auto_close", provider)
}

// TestIo_AppendModeInitialPosition verifies that opening a file in write-only
// append mode ("a") positions the stream at end-of-file, so seek("cur")
// reports the EOF offset (POSIX/glibc and reference Lua behavior). Read+append
// ("a+") deliberately keeps the read position at the start.
func TestIo_AppendModeInitialPosition(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "data.txt"), []byte("ABCDEFGH"), 0644); err != nil {
		t.Fatal(err)
	}
	provider := vm.NewFullIoProvider(tmpDir)
	source := `
		local a = assert(io.open("data.txt", "a"))
		assert(a:seek("cur") == 8, "append 'a' should start at EOF, got " .. a:seek("cur"))
		a:close()

		local ap = assert(io.open("data.txt", "a+"))
		assert(ap:seek("cur") == 0, "'a+' read position should start at 0, got " .. ap:seek("cur"))
		ap:close()

		local w = assert(io.open("data.txt", "a"))
		w:write("IJ"); w:close()
		local r = assert(io.open("data.txt", "r"))
		assert(r:read("a") == "ABCDEFGHIJ", "append should write at EOF")
		r:close()
	`
	runLuaWithIo(t, source, "test_io_append_mode_initial_position", provider)
}
