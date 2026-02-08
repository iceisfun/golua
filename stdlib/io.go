package stdlib

import (
	"fmt"
	"io"
	"strconv"

	"github.com/iceisfun/golua/vm"
)

// fileHandleMeta is a shared metatable for identifying file handles via io.type().
var fileHandleMeta *vm.Table

func init() {
	fileHandleMeta = vm.NewEmptyTable()
	fileHandleMeta.SetString("__name", vm.NewString("FILE*"))
}

// openIo registers the io library if an IoProvider is set.
func openIo(v *vm.VM) {
	provider := v.IoProvider()
	if provider == nil {
		return
	}

	ioTable := vm.NewEmptyTable()
	caps := provider.Capabilities()

	if caps.AllowRead {
		ioTable.SetString("open", vm.NewNativeFunc(makeIoOpen(v, provider)))
		ioTable.SetString("close", vm.NewNativeFunc(ioClose))
		ioTable.SetString("lines", vm.NewNativeFunc(makeIoLines(v, provider)))
	}

	ioTable.SetString("type", vm.NewNativeFunc(ioType))

	v.SetGlobal("io", vm.NewTable(ioTable))
}

// makeFileHandle creates a file handle table wrapping a LuaFile.
func makeFileHandle(f vm.LuaFile) vm.Value {
	handle := vm.NewEmptyTable()
	handle.SetMetatable(fileHandleMeta)

	// Store the Go file as a light userdata-like value via closure capture
	handle.SetString("read", vm.NewNativeFunc(makeFileRead(f)))
	handle.SetString("close", vm.NewNativeFunc(makeFileClose(f)))
	handle.SetString("lines", vm.NewNativeFunc(makeFileLines(f)))

	// Mark as open via a sentinel
	handle.SetString("__file_open", vm.True)

	return vm.NewTable(handle)
}

// markFileClosed marks a file handle as closed.
func markFileClosed(handle vm.LuaTable) {
	handle.Set(vm.NewString("__file_open"), vm.False)
}

// makeIoOpen creates the io.open function.
func makeIoOpen(v *vm.VM, provider vm.LuaIoProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		name := v.Get(1).AsString()
		mode := "r"
		if !v.Get(2).IsNil() {
			mode = v.Get(2).AsString()
		}

		f, err := provider.Open(name, mode)
		if err != nil {
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString(err.Error()))
			return 2
		}

		v.Set(0, makeFileHandle(f))
		return 1
	}
}

// ioClose implements io.close(file).
func ioClose(v *vm.VM) int {
	handle := v.Get(1).AsTable()
	if handle == nil {
		panic("bad argument #1 to 'close' (file expected)")
	}

	closeFunc := handle.Get(vm.NewString("close"))
	if closeFunc.IsNil() || !closeFunc.IsNativeFunc() {
		panic("bad argument #1 to 'close' (file expected)")
	}

	// Call the file's close method
	// We reuse the native function directly
	return closeFunc.AsNativeFunc()(v)
}

// ioType implements io.type(obj).
func ioType(v *vm.VM) int {
	val := v.Get(1)
	if !val.IsTable() {
		v.Set(0, vm.Nil)
		return 1
	}

	t := val.AsTable()
	mt := t.Metatable()
	if mt != fileHandleMeta {
		v.Set(0, vm.Nil)
		return 1
	}

	open := t.Get(vm.NewString("__file_open"))
	if open.IsBool() && open.AsBool() {
		v.Set(0, vm.NewString("file"))
	} else {
		v.Set(0, vm.NewString("closed file"))
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

		// Return an iterator that reads lines and closes the file at the end
		v.Set(0, vm.NewNativeFunc(func(v *vm.VM) int {
			line, err := f.Read("l")
			if err != nil {
				if err == io.EOF {
					f.Close()
					v.Set(0, vm.Nil)
					return 1
				}
				f.Close()
				panic(err.Error())
			}
			v.Set(0, vm.NewString(line))
			return 1
		}))
		return 1
	}
}

// makeFileRead creates the f:read(...) method for a file handle.
// Called as f:read(...), so v.Get(1) is self and real args start at v.Get(2).
func makeFileRead(f vm.LuaFile) vm.NativeFunc {
	return func(v *vm.VM) int {
		n := v.ArgCount() - 1 // subtract self
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

		// Process each format argument (args start at index 2)
		results := 0
		for i := 2; i <= n+1; i++ {
			arg := v.Get(i)
			if arg.IsNumber() {
				// Read N bytes
				count, _ := arg.ToInt()
				data, err := f.ReadBytes(int(count))
				if err != nil {
					v.Set(results, vm.Nil)
				} else {
					v.Set(results, vm.NewString(data))
				}
			} else if arg.IsString() {
				format := arg.AsString()
				data, err := f.Read(format)
				if err != nil {
					v.Set(results, vm.Nil)
				} else {
					if format == "n" || format == "*n" {
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
}

// makeFileClose creates the f:close() method for a file handle.
// Called as f:close(), so v.Get(1) is self.
func makeFileClose(f vm.LuaFile) vm.NativeFunc {
	return func(v *vm.VM) int {
		if f.IsClosed() {
			panic("attempt to close a closed file")
		}

		err := f.Close()
		if err != nil {
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString(err.Error()))
			return 2
		}

		// Mark the handle as closed (for io.type)
		handle := v.Get(1).AsTable()
		if handle != nil {
			markFileClosed(handle)
		}

		v.Set(0, vm.True)
		return 1
	}
}

// makeFileLines creates the f:lines() method for a file handle.
func makeFileLines(f vm.LuaFile) vm.NativeFunc {
	return func(v *vm.VM) int {
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
}
