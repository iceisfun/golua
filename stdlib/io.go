package stdlib

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"syscall"

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
	ioTable.SetString("type", vm.NewNativeFunc(ioType))
	ioTable.SetString("read", vm.NewNativeFunc(makeIoRead(provider)))
	ioTable.SetString("write", vm.NewNativeFunc(makeIoWrite(provider)))

	// Standard file handles
	if f := provider.Stdin(); f != nil {
		ioTable.SetString("stdin", makeFileHandle(f))
	}
	if f := provider.Stdout(); f != nil {
		ioTable.SetString("stdout", makeFileHandle(f))
	}
	if f := provider.Stderr(); f != nil {
		ioTable.SetString("stderr", makeFileHandle(f))
	}

	// io.input() / io.output() default input/output streams
	// We store the current default input/output in the io table itself
	// as __input and __output keys.
	ioVal := vm.NewTable(ioTable)

	// Set defaults to stdin/stdout
	if f := provider.Stdin(); f != nil {
		ioTable.SetString("__input", makeFileHandle(f))
	}
	if f := provider.Stdout(); f != nil {
		ioTable.SetString("__output", makeFileHandle(f))
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

		f, err := provider.Open(nameStr, mode)
		if err != nil {
			v.Set(0, vm.Nil)
			// Extract error message and errno
			errMsg := err.Error()
			var errno int
			if pathErr, ok := err.(*syscall.Errno); ok {
				errno = int(*pathErr)
			} else {
				errno = extractErrno(err)
			}
			v.Set(1, vm.NewString(fmt.Sprintf("%s: %s", nameStr, errMsg)))
			v.Set(2, vm.NewInt(int64(errno)))
			return 3
		}

		v.Set(0, makeFileHandle(f))
		return 1
	}
}

// extractErrno tries to extract an errno from a nested error.
func extractErrno(err error) int {
	// Walk the error chain
	for err != nil {
		if errno, ok := err.(syscall.Errno); ok {
			return int(errno)
		}
		// Try to unwrap
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
		} else {
			break
		}
	}
	// Default errno for generic errors
	return 2 // ENOENT as a reasonable default
}

// makeIoClose creates the io.close([file]) function.
func makeIoClose(provider vm.LuaIoProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		val := v.Get(1)
		if val.IsNil() {
			// Close default output - not supported yet
			panic("cannot close default output file")
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

// makeIoLines creates the io.lines(filename) function.
func makeIoLines(v *vm.VM, provider vm.LuaIoProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		name := v.Get(1).AsString()

		f, err := provider.Open(name, "r")
		if err != nil {
			panic(fmt.Sprintf("cannot open '%s': %s", name, err.Error()))
		}

		closed := false
		// Return an iterator that reads lines and closes the file at the end
		v.Set(0, vm.NewNativeFunc(func(v *vm.VM) int {
			if closed {
				panic("file is already closed")
			}
			line, err := f.Read("l")
			if err != nil {
				if err == io.EOF {
					f.Close()
					closed = true
					v.Set(0, vm.Nil)
					return 1
				}
				f.Close()
				closed = true
				panic(err.Error())
			}
			v.Set(0, vm.NewString(line))
			return 1
		}))
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
				panic(fmt.Sprintf("cannot open '%s': %s", arg.AsString(), err.Error()))
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
				panic(fmt.Sprintf("cannot open '%s': %s", arg.AsString(), err.Error()))
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

// doFileRead performs the actual read operation. firstArg is the index
// of the first format argument (2 for method calls, 1 for io.read).
func doFileRead(v *vm.VM, f vm.LuaFile, firstArg int) int {
	n := v.ArgCount() - (firstArg - 1) // number of format args
	if n <= 0 {
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
	v.EnsureStack(v.Base() + n)
	results := 0
	for i := firstArg; i < firstArg+n; i++ {
		arg := v.Get(i)
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
			data, err := f.Read(format)
			if err != nil {
				v.Set(results, vm.Nil)
			} else {
				// Check if format is "n" or "*n" for number parsing
				cleanFmt := strings.TrimPrefix(format, "*")
				if len(cleanFmt) > 0 && cleanFmt[0] == 'n' {
					// Parse as number
					if fv, err := strconv.ParseFloat(data, 64); err == nil {
						v.Set(results, vm.NewFloat(fv))
					} else {
						v.Set(results, vm.Nil)
					}
				} else {
					v.Set(results, vm.NewString(data))
				}
			}
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
			s = valueToString(arg)
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

// fileLines implements f:lines() method.
func fileLines(v *vm.VM) int {
	fh := getFileHandle(v, v.Get(1), "lines")
	fh.checkOpen("lines")
	f := fh.file

	v.Set(0, vm.NewNativeFunc(func(v *vm.VM) int {
		line, err := f.Read("l")
		if err != nil {
			if err == io.EOF {
				v.Set(0, vm.Nil)
				return 1
			}
			panic(err.Error())
		}
		v.Set(0, vm.NewString(line))
		return 1
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

	var offset int64
	if !v.Get(3).IsNil() {
		var ok bool
		offset, ok = v.Get(3).ToInt()
		if !ok {
			callerArgError(v, 2, "io.seek", "number expected")
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
		callerArgError(v, 1, "io.setvbuf", "string expected, got nil")
	}
	modeStr := mode.AsString()

	var size int
	if !v.Get(3).IsNil() {
		sz, ok := v.Get(3).ToInt()
		if !ok {
			callerArgError(v, 2, "io.setvbuf", "number expected")
		}
		size = int(sz)
	}

	err := fh.file.SetVBuf(modeStr, size)
	if err != nil {
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString(err.Error()))
		return 2
	}

	v.Set(0, self)
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

	v.Set(0, self)
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
