package stdlib

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/iceisfun/golua/vm"
)

// fileHandleMeta is a shared metatable for file handle userdata.
// It contains __index pointing to a methods table, __name, __tostring,
// and __gc for identification and method dispatch.
var fileHandleMeta *vm.Table

// fileMethodsTable contains the methods available on file handles.
var fileMethodsTable *vm.Table

func init() {
	fileMethodsTable = vm.NewEmptyTable()
	// Populate file methods. These are instance methods dispatched via __index.
	// Each method extracts the LuaFile from the userdata's fileHandle.
	fileMethodsTable.SetString("read", vm.NewNativeFunc(fileRead))
	fileMethodsTable.SetString("close", vm.NewNativeFunc(fileClose))
	fileMethodsTable.SetString("lines", vm.NewNativeFunc(fileLines))
	fileMethodsTable.SetString("write", vm.NewNativeFunc(fileWrite))
	fileMethodsTable.SetString("seek", vm.NewNativeFunc(fileSeek))
	fileMethodsTable.SetString("setvbuf", vm.NewNativeFunc(fileSetVBuf))
	fileMethodsTable.SetString("flush", vm.NewNativeFunc(fileFlush))

	fileHandleMeta = vm.NewEmptyTable()
	fileHandleMeta.SetString("__name", vm.NewString("FILE*"))
	fileHandleMeta.SetString("__index", vm.NewTable(fileMethodsTable))
	fileHandleMeta.SetString("__tostring", vm.NewNativeFunc(fileToString))
	// __close and __gc silently close the file if not already closed.
	// This is needed for to-be-closed variables (generic for with io.lines).
	closeGC := vm.NewNativeFunc(fileCloseGC)
	fileHandleMeta.SetString("__close", closeGC)
	fileHandleMeta.SetString("__gc", closeGC)
}

// fileHandle is the Go data stored inside a file userdata value.
type fileHandle struct {
	file   vm.LuaFile
	closed bool
}

// makeFileHandle creates a file handle userdata wrapping a LuaFile.
func makeFileHandle(f vm.LuaFile) vm.Value {
	fh := &fileHandle{file: f}
	return vm.NewUserdataValue(fh, fileHandleMeta)
}

// fileMethodArgError raises a "bad argument" error for file methods.
// Unlike callerArgError, this does not adjust the arg index for method calls
// and uses '?' as the function name, matching Lua 5.4 behavior for file methods
// dispatched via __index.
func fileMethodArgError(v *vm.VM, idx int, fallback, msg string) {
	name, _ := v.CallerFuncName()
	if name == "" {
		name = fallback
	}
	panic(fmt.Sprintf("bad argument #%d to '%s' (%s)", idx, name, msg))
}

// getFileHandle extracts the fileHandle from a userdata value, or panics.
func getFileHandle(v *vm.VM, val vm.Value, funcName string) *fileHandle {
	ud := val.AsUserdata()
	if ud == nil {
		callerArgError(v, 1, funcName, fmt.Sprintf("FILE* expected, got %s", v.ObjTypeName(val)))
	}
	fh, ok := ud.Data.(*fileHandle)
	if !ok {
		callerArgError(v, 1, funcName, "FILE* expected")
	}
	return fh
}

// checkOpen panics if the file handle is closed.
func (fh *fileHandle) checkOpen(method string) {
	if fh.closed || fh.file.IsClosed() {
		panic(fmt.Sprintf("attempt to use a closed file"))
	}
}

// openIo registers the io library if an IoProvider is set.
func openIo(v *vm.VM) {
	provider := v.IoProvider()
	if provider == nil {
		return
	}

	// Build the io table
	ioTable := vm.NewEmptyTable()
	caps := provider.Capabilities()

	if caps.AllowRead || caps.AllowWrite {
		ioTable.SetString("open", vm.NewNativeFunc(makeIoOpen(v, provider)))
		ioTable.SetString("lines", vm.NewNativeFunc(makeIoLines(v, provider)))
	}

	ioTable.SetString("close", vm.NewNativeFunc(makeIoClose(provider)))
	if caps.AllowWrite {
		ioTable.SetString("flush", vm.NewNativeFunc(makeIoFlush(provider)))
	}
	ioTable.SetString("type", vm.NewNativeFunc(ioType))
	ioTable.SetString("read", vm.NewNativeFunc(makeIoRead(provider)))
	ioTable.SetString("write", vm.NewNativeFunc(makeIoWrite(provider)))
	ioTable.SetString("tmpfile", vm.NewNativeFunc(makeIoTmpfile(provider)))

	// Standard file handles - create once and share between io.stdin/io.stdout/io.stderr
	// and __input/__output so identity comparisons work (io.input() == io.stdin).
	var stdinHandle, stdoutHandle, stderrHandle vm.Value
	if f := provider.Stdin(); f != nil {
		stdinHandle = makeFileHandle(f)
		ioTable.SetString("stdin", stdinHandle)
	}
	if f := provider.Stdout(); f != nil {
		stdoutHandle = makeFileHandle(f)
		ioTable.SetString("stdout", stdoutHandle)
	}
	if f := provider.Stderr(); f != nil {
		stderrHandle = makeFileHandle(f)
		ioTable.SetString("stderr", stderrHandle)
	}

	// io.input() / io.output() default input/output streams
	// We store the current default input/output in the io table itself
	// as __input and __output keys. These share identity with io.stdin/io.stdout.
	ioVal := vm.NewTable(ioTable)

	// Set defaults to the same handle objects as stdin/stdout
	if !stdinHandle.IsNil() {
		ioTable.SetString("__input", stdinHandle)
	}
	if !stdoutHandle.IsNil() {
		ioTable.SetString("__output", stdoutHandle)
	}

	ioTable.SetString("input", vm.NewNativeFunc(makeIoInput(v, provider, ioTable)))
	ioTable.SetString("output", vm.NewNativeFunc(makeIoOutput(v, provider, ioTable)))

	v.SetGlobal("io", ioVal)
}

// makeIoOpen creates the io.open function.
func makeIoOpen(v *vm.VM, provider vm.LuaIoProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		name := v.Get(1)
		if name.IsNil() {
			callerArgError(v, 1, "io.open", "string expected, got nil")
		}
		nameStr := name.AsString()
		mode := "r"
		if !v.Get(2).IsNil() {
			mode = v.Get(2).AsString()
		}

		// Validate mode before calling provider (invalid mode is a hard error)
		cleanMode := strings.TrimSuffix(mode, "b")
		switch cleanMode {
		case "r", "w", "a", "r+", "w+", "a+":
			// valid
		default:
			callerArgError(v, 2, "io.open", "invalid mode")
		}

		f, err := provider.Open(nameStr, mode)
		if err != nil {
			v.Set(0, vm.Nil)
			msg, errno := formatOpenError(nameStr, err)
			v.Set(1, vm.NewString(msg))
			v.Set(2, vm.NewInt(int64(errno)))
			return 3
		}

		v.Set(0, makeFileHandle(f))
		return 1
	}
}

// formatOpenError formats an io.open error for Lua.
// Go's os.OpenFile returns "*os.PathError" with format "open /path: error".
// Lua expects "/path: Error description" with errno.
func formatOpenError(name string, err error) (string, int) {
	return formatPathError(name, err)
}

// extractLuaFileError extracts a human-readable error description from a Go error,
// stripping the operation prefix. Returns (errno, description).
func extractLuaFileError(err error) (int, string) {
	_, errno := formatPathError("", err)
	// Get just the error description, capitalized
	msg := err.Error()
	// Try to extract the inner error from PathError
	if pe, ok := err.(*os.PathError); ok {
		msg = capitalizeError(pe.Err.Error())
	} else if le, ok := err.(*os.LinkError); ok {
		msg = capitalizeError(le.Err.Error())
	} else {
		msg = capitalizeError(msg)
	}
	return errno, msg
}

// makeIoTmpfile creates the io.tmpfile() function.
func makeIoTmpfile(provider vm.LuaIoProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		f, err := provider.TmpFile()
		if err != nil {
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString(err.Error()))
			return 2
		}
		v.Set(0, makeFileHandle(f))
		return 1
	}
}

// makeIoClose creates the io.close([file]) function.
func makeIoClose(provider vm.LuaIoProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		val := v.Get(1)
		if val.IsNil() {
			// Close default output - standard files cannot be closed
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString("cannot close standard file"))
			return 2
		}

		fh := getFileHandle(v, val, "io.close")
		if fh.file.IsStd() {
			// Cannot close standard files - return nil, error
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString("cannot close standard file"))
			return 2
		}
		if fh.closed || fh.file.IsClosed() {
			panic("attempt to use a closed file")
		}

		err := fh.file.Close()
		fh.closed = true
		if err != nil {
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString(err.Error()))
			return 2
		}

		v.Set(0, vm.True)
		return 1
	}
}

// ioType implements io.type(obj).
func ioType(v *vm.VM) int {
	if v.ArgCount() < 1 {
		callerArgError(v, 1, "io.type", "value expected")
	}
	val := v.Get(1)
	ud := val.AsUserdata()
	if ud == nil {
		v.Set(0, vm.Nil)
		return 1
	}

	fh, ok := ud.Data.(*fileHandle)
	if !ok {
		v.Set(0, vm.Nil)
		return 1
	}

	if fh.closed || fh.file.IsClosed() {
		v.Set(0, vm.NewString("closed file"))
	} else {
		v.Set(0, vm.NewString("file"))
	}
	return 1
}

// makeIoFlush creates the io.flush() function that flushes default output.
func makeIoFlush(provider vm.LuaIoProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		ioVal := v.GetGlobal("io")
		if ioVal.IsNil() {
			panic("io library not available")
		}
		ioTable := ioVal.AsTable()
		outputVal := ioTable.Get(vm.NewString("__output"))
		fh := getFileHandle(v, outputVal, "io.flush")
		fh.checkOpen("flush")

		if err := fh.file.Flush(); err != nil {
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString(err.Error()))
			return 2
		}

		v.Set(0, vm.True)
		return 1
	}
}

// makeIoLines creates the io.lines(filename, ...) function.
func makeIoLines(v *vm.VM, provider vm.LuaIoProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		var f vm.LuaFile
		var toClose bool
		var formats []vm.Value

		if v.ArgCount() == 0 || v.Get(1).IsNil() {
			// Read from default input
			ioVal := v.GetGlobal("io")
			if ioVal.IsNil() {
				panic("io library not available")
			}
			ioTable := ioVal.AsTable()
			inputVal := ioTable.Get(vm.NewString("__input"))
			fh := getFileHandle(v, inputVal, "io.lines")
			fh.checkOpen("lines")
			f = fh.file
			toClose = false
		} else {
			name := v.Get(1).AsString()
			var err error
			f, err = provider.Open(name, "r")
			if err != nil {
				_, errDesc := extractLuaFileError(err)
				panic(fmt.Sprintf("cannot open file '%s' (%s)", name, errDesc))
			}
			toClose = true
		}
		for i := 2; i <= v.ArgCount(); i++ {
			formats = append(formats, v.Get(i))
		}

		closed := false
		// Return an iterator that reads lines
		v.Set(0, vm.NewNativeFunc(func(v *vm.VM) int {
			if closed {
				panic("file is already closed")
			}

			if len(formats) == 0 {
				line, err := f.Read("l")
				if err != nil {
					if err == io.EOF {
						if toClose {
							f.Close()
							closed = true
						}
						// Return no results on EOF
						return 0
					}
					panic(err.Error())
				}
				v.Set(0, vm.NewString(line))
				return 1
			}

			results := doFileReadFormats(v, f, formats)
			if results > 0 && v.Get(0).IsNil() {
				if toClose {
					f.Close()
					closed = true
				}
				// Return no results on EOF
				return 0
			}
			return results
		}))
		if toClose {
			// Return iterator, nil, nil, filehandle (4 values)
			// The 4th value is a to-be-closed variable for generic for
			v.Set(1, vm.Nil)
			v.Set(2, vm.Nil)
			v.Set(3, makeFileHandle(f))
			return 4
		}
		return 1
	}
}

// makeIoInput creates the io.input([file]) function.
func makeIoInput(vmRef *vm.VM, provider vm.LuaIoProvider, ioTable *vm.Table) vm.NativeFunc {
	return func(v *vm.VM) int {
		arg := v.Get(1)
		if arg.IsNil() {
			// Return current default input
			v.Set(0, ioTable.Get(vm.NewString("__input")))
			return 1
		}

		if arg.IsString() {
			// Open file as default input
			f, err := provider.Open(arg.AsString(), "r")
			if err != nil {
				_, errDesc := extractLuaFileError(err)
				panic(fmt.Sprintf("cannot open file '%s' (%s)", arg.AsString(), errDesc))
			}
			handle := makeFileHandle(f)
			ioTable.SetString("__input", handle)
			v.Set(0, handle)
			return 1
		}

		// Assume it's a file handle - set as default input
		_ = getFileHandle(v, arg, "io.input") // validate it's a file handle
		ioTable.SetString("__input", arg)
		v.Set(0, arg)
		return 1
	}
}

// makeIoOutput creates the io.output([file]) function.
func makeIoOutput(vmRef *vm.VM, provider vm.LuaIoProvider, ioTable *vm.Table) vm.NativeFunc {
	return func(v *vm.VM) int {
		arg := v.Get(1)
		if arg.IsNil() {
			// Return current default output
			v.Set(0, ioTable.Get(vm.NewString("__output")))
			return 1
		}

		if arg.IsString() {
			// Open file as default output
			f, err := provider.Open(arg.AsString(), "w")
			if err != nil {
				_, errDesc := extractLuaFileError(err)
				panic(fmt.Sprintf("cannot open file '%s' (%s)", arg.AsString(), errDesc))
			}
			handle := makeFileHandle(f)
			ioTable.SetString("__output", handle)
			v.Set(0, handle)
			return 1
		}

		// Assume it's a file handle - set as default output
		_ = getFileHandle(v, arg, "io.output") // validate it's a file handle
		ioTable.SetString("__output", arg)
		v.Set(0, arg)
		return 1
	}
}

// makeIoRead creates the io.read(...) function that reads from default input.
func makeIoRead(provider vm.LuaIoProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		// Get io table to find default input
		ioVal := v.GetGlobal("io")
		if ioVal.IsNil() {
			panic("io library not available")
		}
		ioTable := ioVal.AsTable()
		inputVal := ioTable.Get(vm.NewString("__input"))
		fh := getFileHandle(v, inputVal, "io.read")
		fh.checkOpen("read")

		return doFileRead(v, fh.file, 1)
	}
}

// makeIoWrite creates the io.write(...) function that writes to default output.
func makeIoWrite(provider vm.LuaIoProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		// Get io table to find default output
		ioVal := v.GetGlobal("io")
		if ioVal.IsNil() {
			panic("io library not available")
		}
		ioTable := ioVal.AsTable()
		outputVal := ioTable.Get(vm.NewString("__output"))
		fh := getFileHandle(v, outputVal, "io.write")
		fh.checkOpen("write")

		return doFileWrite(v, fh.file, outputVal, 1)
	}
}

// fileRead implements f:read(...) method.
func fileRead(v *vm.VM) int {
	fh := getFileHandle(v, v.Get(1), "read")
	fh.checkOpen("read")

	return doFileRead(v, fh.file, 2)
}

// doFileRead performs the actual read operation using arguments from the stack.
// firstArg is the index of the first format argument (2 for method calls, 1 for io.read).
func doFileRead(v *vm.VM, f vm.LuaFile, firstArg int) int {
	n := v.ArgCount() - (firstArg - 1) // number of format args
	var formats []vm.Value
	if n > 0 {
		formats = make([]vm.Value, n)
		for i := 0; i < n; i++ {
			formats[i] = v.Get(firstArg + i)
		}
	}
	return doFileReadFormats(v, f, formats)
}

// doFileReadFormats performs the actual read operation using a slice of formats.
func doFileReadFormats(v *vm.VM, f vm.LuaFile, formats []vm.Value) int {
	if len(formats) == 0 {
		// Default: read a line
		line, err := f.Read("l")
		if err != nil {
			if err == io.EOF {
				v.Set(0, vm.Nil)
				return 1
			}
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString(err.Error()))
			return 2
		}
		v.Set(0, vm.NewString(line))
		return 1
	}

	// Process each format argument
	n := len(formats)
	v.EnsureStack(v.Base() + n)
	results := 0
	for _, arg := range formats {
		if arg.IsNumber() {
			// Read N bytes
			count, _ := arg.ToInt()
			if count == 0 {
				// Read 0 bytes: test if at EOF
				data, err := f.ReadBytes(0)
				if err != nil {
					v.Set(results, vm.Nil)
				} else {
					v.Set(results, vm.NewString(data))
				}
			} else {
				data, err := f.ReadBytes(int(count))
				if err != nil {
					v.Set(results, vm.Nil)
				} else {
					v.Set(results, vm.NewString(data))
				}
			}
		} else if arg.IsString() {
			format := arg.AsString()
			// Validate format before calling provider
			cleanFmt := strings.TrimPrefix(format, "*")
			if len(cleanFmt) == 0 || (cleanFmt[0] != 'a' && cleanFmt[0] != 'l' && cleanFmt[0] != 'L' && cleanFmt[0] != 'n') {
				// Use results+1 as the argument index (1-based, for the format arg)
				callerArgError(v, results+1, "read", "invalid format")
			}
			data, err := f.Read(format)
			if err != nil {
				v.Set(results, vm.Nil)
			} else {
				// Check if format is "n" or "*n" for number parsing
				cleanFmt := strings.TrimPrefix(format, "*")
				if len(cleanFmt) > 0 && cleanFmt[0] == 'n' {
					// Parse as number: try integer first, then float
					if iv, err := strconv.ParseInt(data, 0, 64); err == nil {
						v.Set(results, vm.NewInt(iv))
					} else if fv, err := strconv.ParseFloat(data, 64); err == nil {
						// Check if float has integer value
						iv := int64(fv)
						if fv == float64(iv) && !strings.ContainsAny(data, ".eE") {
							v.Set(results, vm.NewInt(iv))
						} else {
							v.Set(results, vm.NewFloat(fv))
						}
					} else {
						v.Set(results, vm.Nil)
					}
				} else {
					v.Set(results, vm.NewString(data))
				}
			}
		} else {
			// Invalid format type
			fileMethodArgError(v, results+1, "read", fmt.Sprintf("string expected, got %s", arg.Type()))
		}
		results++
	}
	return results
}

// fileWrite implements f:write(...) method.
func fileWrite(v *vm.VM) int {
	self := v.Get(1)
	fh := getFileHandle(v, self, "write")
	fh.checkOpen("write")

	return doFileWrite(v, fh.file, self, 2)
}

// doFileWrite performs the actual write operation. firstArg is the index
// of the first data argument.
func doFileWrite(v *vm.VM, f vm.LuaFile, self vm.Value, firstArg int) int {
	n := v.ArgCount() - (firstArg - 1)
	for i := firstArg; i < firstArg+n; i++ {
		arg := v.Get(i)
		var s string
		if arg.IsString() {
			s = arg.AsString()
		} else if arg.IsNumber() {
			// io.write uses C's fprintf with %.14g for floats, which does
			// NOT append ".0" to integer-valued floats (unlike tostring()).
			s = valueToString(arg)
			if !arg.IsInt() && strings.HasSuffix(s, ".0") {
				before := s[:len(s)-2]
				if !strings.ContainsAny(before, ".eE") {
					s = before
				}
			}
		} else {
			callerArgError(v, i-firstArg+1, "io.write", fmt.Sprintf("string expected, got %s", arg.Type()))
		}
		err := f.Write(s)
		if err != nil {
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString(err.Error()))
			return 2
		}
	}

	// Return the file handle (for chaining)
	v.Set(0, self)
	return 1
}

// fileClose implements f:close() method.
func fileClose(v *vm.VM) int {
	self := v.Get(1)
	fh := getFileHandle(v, self, "close")

	if fh.file.IsStd() {
		// Cannot close standard files
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString("cannot close standard file"))
		return 2
	}

	if fh.closed || fh.file.IsClosed() {
		panic("attempt to use a closed file")
	}

	err := fh.file.Close()
	fh.closed = true
	if err != nil {
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString(err.Error()))
		return 2
	}

	v.Set(0, vm.True)
	return 1
}

// fileLines implements f:lines(...) method.
func fileLines(v *vm.VM) int {
	fh := getFileHandle(v, v.Get(1), "lines")
	fh.checkOpen("lines")
	f := fh.file

	var formats []vm.Value
	for i := 2; i <= v.ArgCount(); i++ {
		formats = append(formats, v.Get(i))
	}

	v.Set(0, vm.NewNativeFunc(func(v *vm.VM) int {
		fh.checkOpen("lines iterator")

		if len(formats) == 0 {
			line, err := f.Read("l")
			if err != nil {
				if err == io.EOF {
					return 0
				}
				panic(err.Error())
			}
			v.Set(0, vm.NewString(line))
			return 1
		}

		results := doFileReadFormats(v, f, formats)
		if results > 0 && v.Get(0).IsNil() {
			return 0
		}
		return results
	}))
	return 1
}

// fileSeek implements f:seek([whence [, offset]]) method.
func fileSeek(v *vm.VM) int {
	fh := getFileHandle(v, v.Get(1), "seek")
	fh.checkOpen("seek")

	whence := "cur"
	if !v.Get(2).IsNil() {
		whence = v.Get(2).AsString()
	}

	// Validate whence before calling provider (invalid whence is a hard error)
	switch whence {
	case "set", "cur", "end":
		// valid
	default:
		fileMethodArgError(v, 2, "seek", fmt.Sprintf("invalid option '%s'", whence))
	}

	var offset int64
	if !v.Get(3).IsNil() {
		var ok bool
		offset, ok = v.Get(3).ToInt()
		if !ok {
			fileMethodArgError(v, 3, "seek", "number expected")
		}
	}

	pos, err := fh.file.Seek(whence, offset)
	if err != nil {
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString(err.Error()))
		return 2
	}

	v.Set(0, vm.NewInt(pos))
	return 1
}

// fileSetVBuf implements f:setvbuf(mode [, size]) method.
func fileSetVBuf(v *vm.VM) int {
	self := v.Get(1)
	fh := getFileHandle(v, self, "setvbuf")
	fh.checkOpen("setvbuf")

	mode := v.Get(2)
	if mode.IsNil() {
		fileMethodArgError(v, 1, "setvbuf", "string expected, got nil")
	}
	modeStr := mode.AsString()

	var size int
	if !v.Get(3).IsNil() {
		sz, ok := v.Get(3).ToInt()
		if !ok {
			fileMethodArgError(v, 3, "setvbuf", "number expected")
		}
		size = int(sz)
	}

	// Validate mode before calling provider (invalid mode is a hard error)
	switch modeStr {
	case "no", "full", "line":
		// valid
	default:
		fileMethodArgError(v, 2, "setvbuf", fmt.Sprintf("invalid option '%s'", modeStr))
	}

	err := fh.file.SetVBuf(modeStr, size)
	if err != nil {
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString(err.Error()))
		return 2
	}

	// Lua 5.4: setvbuf returns true on success (not the file handle)
	v.Set(0, vm.True)
	return 1
}

// fileFlush implements f:flush() method.
func fileFlush(v *vm.VM) int {
	self := v.Get(1)
	fh := getFileHandle(v, self, "flush")
	fh.checkOpen("flush")

	err := fh.file.Flush()
	if err != nil {
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString(err.Error()))
		return 2
	}

	v.Set(0, vm.True)
	return 1
}

// fileToString implements __tostring for file handles.
func fileToString(v *vm.VM) int {
	val := v.Get(1)
	ud := val.AsUserdata()
	if ud == nil {
		v.Set(0, vm.NewString("file (?)"))
		return 1
	}
	fh, ok := ud.Data.(*fileHandle)
	if !ok {
		v.Set(0, vm.NewString("file (?)"))
		return 1
	}
	if fh.closed || fh.file.IsClosed() {
		v.Set(0, vm.NewString("file (closed)"))
	} else {
		v.Set(0, vm.NewString(fmt.Sprintf("file (%p)", ud)))
	}
	return 1
}

// fileCloseGC implements __close and __gc for file handles.
// Unlike fileClose, this silently handles already-closed and standard files.
func fileCloseGC(v *vm.VM) int {
	val := v.Get(1)
	ud := val.AsUserdata()
	if ud == nil {
		return 0
	}
	fh, ok := ud.Data.(*fileHandle)
	if !ok {
		return 0
	}
	if fh.closed || fh.file.IsClosed() || fh.file.IsStd() {
		return 0
	}
	fh.file.Close()
	fh.closed = true
	return 0
}
