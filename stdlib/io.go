package stdlib

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/iceisfun/golua/v2/vm"
)

// maxArgLine is the maximum number of format arguments for f:lines()/io.lines().
// Matches Lua 5.4's MAXARGLINE (liolib.c).
const maxArgLine = 250

// Internal io-module table keys storing the current default input/output file
// handles (io.input()/io.output()). These are private storage slots, not Lua
// metamethods, so they carry the "__" prefix to stay hidden from scripts.
const (
	ioFieldInput  = "__input"
	ioFieldOutput = "__output"
)

// file:seek whence values (Lua §6.8).
const (
	seekSet = "set"
	seekCur = "cur"
	seekEnd = "end"
)

// file:setvbuf buffering modes (Lua §6.8).
const (
	vbufNo   = "no"
	vbufFull = "full"
	vbufLine = "line"
)

// ioState holds per-VM file handle tables created when the IO provider is first used.
type ioState struct {
	meta    *vm.Table // metatable for file handle userdata
	methods *vm.Table // methods table (__index target)
}

// getIoState returns the per-VM ioState, creating it on first call.
func getIoState(v *vm.VM) *ioState {
	if s := v.InternalState("io"); s != nil {
		return s.(*ioState)
	}

	methods := vm.NewEmptyTable()
	methods.SetString("read", vm.NewNativeFunc(fileRead))
	methods.SetString("close", vm.NewNativeFunc(fileClose))
	methods.SetString("lines", vm.NewNativeFunc(fileLines))
	methods.SetString("write", vm.NewNativeFunc(fileWrite))
	methods.SetString("seek", vm.NewNativeFunc(fileSeek))
	methods.SetString("setvbuf", vm.NewNativeFunc(fileSetVBuf))
	methods.SetString("flush", vm.NewNativeFunc(fileFlush))

	meta := vm.NewEmptyTable()
	meta.SetString(vm.MetaName, vm.NewString("FILE*"))
	meta.SetString(vm.MetaIndex, vm.NewTable(methods))
	meta.SetString(vm.MetaTostring, vm.NewNativeFunc(fileToString))
	// __close and __gc silently close the file if not already closed.
	// This is needed for to-be-closed variables (generic for with io.lines).
	closeGC := vm.NewNativeFunc(fileCloseGC)
	meta.SetString(vm.MetaClose, closeGC)
	meta.SetString(vm.MetaGC, closeGC)

	s := &ioState{meta: meta, methods: methods}
	v.SetInternalState("io", s)
	return s
}

// fileHandle is the Go data stored inside a file userdata value.
type fileHandle struct {
	file      vm.LuaFile
	closed    bool
	closeFn   func(*vm.VM, *fileHandle) int
	gcCloseFn func(*fileHandle)
}

// makeFileHandle creates a file handle userdata wrapping a LuaFile.
func makeFileHandle(v *vm.VM, f vm.LuaFile) vm.Value {
	fh := &fileHandle{file: f}
	val := vm.NewUserdataValueUV(fh, getIoState(v).meta, 0)
	// Register the __gc finalizer so an unreferenced (never explicitly closed)
	// file handle is closed — and its buffered writes flushed — when collected,
	// matching reference Lua's file __gc behavior.
	v.RegisterGcFinalizerUserdata(val.AsUserdata())
	return val
}

func makeFileHandleWithClose(v *vm.VM, f vm.LuaFile, closeFn func(*vm.VM, *fileHandle) int, gcCloseFn func(*fileHandle)) vm.Value {
	fh := &fileHandle{file: f, closeFn: closeFn, gcCloseFn: gcCloseFn}
	val := vm.NewUserdataValueUV(fh, getIoState(v).meta, 0)
	v.RegisterGcFinalizerUserdata(val.AsUserdata())
	return val
}

// fileArgError raises a "bad argument" error for file operations.
// It mirrors Lua 5.4's luaL_argerror: resolves the function name from the
// caller's bytecode, and if the call was via method syntax (OP_SELF),
// decrements the arg number by 1 (since self is implicit).
// idx is the 1-based argument position counting self (e.g. 2 for the first
// explicit arg in a method call, 1 for a module function call).
// fallback is used if the name cannot be resolved from bytecode.
func fileArgError(v *vm.VM, idx int, _ string, msg string) {
	name, nameWhat := v.ArgErrorFuncName()
	if nameWhat == "method" {
		idx-- // method call: self is implicit, decrement arg number
	}
	panic(fmt.Sprintf("bad argument #%d to '%s' (%s)", idx, name, msg))
}

// getFileHandle extracts the fileHandle from a userdata value, or panics.
func getFileHandle(v *vm.VM, val vm.Value, funcName string) *fileHandle {
	ud := val.AsUserdata()
	if ud == nil {
		var gotStr string
		if v.ArgCount() < 1 {
			gotStr = "got no value"
		} else {
			gotStr = fmt.Sprintf("got %s", v.ObjTypeName(val))
		}
		callerArgError(v, 1, funcName, fmt.Sprintf("FILE* expected, %s", gotStr))
	}
	fh, ok := ud.Data.(*fileHandle)
	if !ok {
		callerArgError(v, 1, funcName, "FILE* expected")
	}
	return fh
}

// checkOpen panics if the file handle is closed.
func (fh *fileHandle) checkOpen(ctx context.Context, method string) {
	if fh.closed || fh.file.IsClosed(ctx) {
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
	ctx := v.Context()
	caps := provider.Capabilities(ctx)

	if caps.AllowRead || caps.AllowWrite {
		ioTable.SetString("open", vm.NewNativeFunc(makeIoOpen(v, provider)))
		ioTable.SetString("lines", vm.NewNativeFunc(makeIoLines(v, provider)))
	}
	if v.ProcessProvider() != nil {
		ioTable.SetString("popen", vm.NewNativeFunc(makeIoPopen(v.ProcessProvider())))
	}

	ioTable.SetString("close", vm.NewNativeFunc(makeIoClose(ioTable)))
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
	if f := provider.Stdin(ctx); f != nil {
		stdinHandle = makeFileHandle(v, f)
		ioTable.SetString("stdin", stdinHandle)
	}
	if f := provider.Stdout(ctx); f != nil {
		stdoutHandle = makeFileHandle(v, f)
		ioTable.SetString("stdout", stdoutHandle)
	}
	if f := provider.Stderr(ctx); f != nil {
		stderrHandle = makeFileHandle(v, f)
		ioTable.SetString("stderr", stderrHandle)
	}

	// io.input() / io.output() default input/output streams
	// We store the current default input/output in the io table itself
	// as __input and __output keys. These share identity with io.stdin/io.stdout.
	ioVal := vm.NewTable(ioTable)

	// Set defaults to the same handle objects as stdin/stdout
	if !stdinHandle.IsNil() {
		ioTable.SetString(ioFieldInput, stdinHandle)
	}
	if !stdoutHandle.IsNil() {
		ioTable.SetString(ioFieldOutput, stdoutHandle)
	}

	ioTable.SetString("input", vm.NewNativeFunc(makeIoInput(v, provider, ioTable)))
	ioTable.SetString("output", vm.NewNativeFunc(makeIoOutput(v, provider, ioTable)))

	v.SetGlobal("io", ioVal)
}

// readNumberFromReader uses a state-machine scanner matching Lua 5.4's
// read_number (liolib.c). On failure, all scanned characters are consumed.
func readNumberFromReader(reader *bufio.Reader) (string, error) {
	// Skip leading whitespace
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return "", err
		}
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			_ = reader.UnreadByte()
			break
		}
	}

	const maxNumLen = 200 // Lua 5.4's L_MAXLENNUM

	offset := 0
	tooLong := false

	peekByte := func() (byte, bool) {
		if offset >= maxNumLen {
			tooLong = true
			return 0, false
		}
		peeked, _ := reader.Peek(offset + 1)
		if len(peeked) <= offset {
			return 0, false
		}
		return peeked[offset], true
	}
	accept := func() { offset++ }
	isDigit := func(b byte) bool { return b >= '0' && b <= '9' }
	isHexDigit := func(b byte) bool {
		return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
	}
	fail := func() (string, error) {
		if offset > 0 {
			reader.Discard(offset)
		}
		return "", fmt.Errorf("not a number")
	}

	// Optional sign
	if b, ok := peekByte(); ok && (b == '+' || b == '-') {
		accept()
	}

	// Determine if hex (0x/0X prefix)
	isHex := false
	hasDigits := false
	if b, ok := peekByte(); ok && b == '0' {
		accept()
		hasDigits = true // the '0' itself is a valid digit
		if b2, ok2 := peekByte(); ok2 && (b2 == 'x' || b2 == 'X') {
			accept()
			isHex = true
			hasDigits = false // need at least one hex digit after 0x
		}
	}

	digitCheck := isDigit
	if isHex {
		digitCheck = isHexDigit
	}

	for {
		b, ok := peekByte()
		if !ok || !digitCheck(b) {
			break
		}
		accept()
		hasDigits = true
	}

	if b, ok := peekByte(); ok && b == '.' {
		accept()
		for {
			b, ok := peekByte()
			if !ok || !digitCheck(b) {
				break
			}
			accept()
			hasDigits = true
		}
	}

	if !hasDigits {
		return fail()
	}

	expChar, expCharU := byte('e'), byte('E')
	if isHex {
		expChar, expCharU = 'p', 'P'
	}
	if b, ok := peekByte(); ok && (b == expChar || b == expCharU) {
		accept()
		if b2, ok2 := peekByte(); ok2 && (b2 == '+' || b2 == '-') {
			accept()
		}
		hasExpDigits := false
		for {
			b, ok := peekByte()
			if !ok || !isDigit(b) {
				break
			}
			accept()
			hasExpDigits = true
		}
		if !hasExpDigits {
			return fail()
		}
	}

	// If the number exceeded the maximum length, it's invalid.
	if tooLong {
		return fail()
	}

	peeked, _ := reader.Peek(offset)
	result := string(peeked[:offset])
	reader.Discard(offset)
	return result, nil
}

// makeIoOpen creates the io.open function.
func makeIoOpen(v *vm.VM, provider vm.LuaIoProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		name := v.Get(1)
		if name.IsNil() {
			if v.ArgCount() < 1 {
				callerArgError(v, 1, "io.open", "string expected, got no value")
			}
			callerArgError(v, 1, "io.open", "string expected, got nil")
		}
		var nameStr string
		if name.IsString() {
			nameStr = name.AsString()
		} else if name.IsNumber() {
			nameStr = vm.ValueToString(name)
		} else {
			callerArgError(v, 1, "io.open", fmt.Sprintf("string expected, got %s", v.ObjTypeName(name)))
		}
		mode := "r"
		if modeArg := v.Get(2); !modeArg.IsNil() {
			if !modeArg.IsString() {
				callerArgError(v, 2, "io.open", fmt.Sprintf("string expected, got %s", v.ObjTypeName(modeArg)))
			}
			mode = modeArg.AsString()
		}

		// Validate mode before calling provider (invalid mode is a hard error).
		// Reference Lua's l_checkmode parses mode as r/w/a, optional '+', then
		// any number of trailing 'b' bytes.
		cleanMode := strings.TrimRight(mode, "b")
		switch cleanMode {
		case "r", "w", "a", "r+", "w+", "a+":
			// valid
		default:
			callerArgError(v, 2, "io.open", "invalid mode")
		}

		ctx := v.Context()
		f, err := provider.Open(ctx, nameStr, mode)
		if err != nil {
			v.Set(0, vm.Nil)
			msg, errno := formatOpenError(nameStr, err)
			v.Set(1, vm.NewString(msg))
			v.Set(2, vm.NewInt(int64(errno)))
			return 3
		}

		v.Set(0, makeFileHandle(v, f))
		return 1
	}
}

// formatOpenError formats an io.open error for Lua.
// Go's os.OpenFile returns "*os.PathError" with format "open /path: error".
// Lua expects "/path: Error description" with errno.
func formatOpenError(name string, err error) (string, int) {
	return formatPathError(name, err)
}

// isFileError reports whether err is an OS-level I/O error (like "bad file descriptor")
// as opposed to a normal read condition (EOF, no number found, etc.).
// Lua 5.4 uses ferror(f) to distinguish these cases.
func isFileError(err error) bool {
	if err == nil || err == io.EOF {
		return false
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return true
	}
	var sysErr syscall.Errno
	if errors.As(err, &sysErr) {
		return true
	}
	return false
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
		ctx := v.Context()
		f, err := provider.TmpFile(ctx)
		if err != nil {
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString(err.Error()))
			return 2
		}
		v.Set(0, makeFileHandle(v, f))
		return 1
	}
}

// makeIoClose creates the io.close([file]) function.
func makeIoClose(ioTable *vm.Table) vm.NativeFunc {
	return func(v *vm.VM) int {
		val := v.Get(1)
		if v.ArgCount() == 0 {
			// io.close() with no args: close default output
			val = ioTable.Get(vm.NewString(ioFieldOutput))
			if val.IsNil() {
				v.Set(0, vm.Nil)
				v.Set(1, vm.NewString("cannot close standard file"))
				return 2
			}
		}

		fh := getFileHandle(v, val, "io.close")
		if fh.closeFn != nil {
			return fh.closeFn(v, fh)
		}
		ctx := v.Context()
		if fh.file.IsStd(ctx) {
			// Cannot close standard files - return nil, error
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString("cannot close standard file"))
			return 2
		}
		if fh.closed || fh.file.IsClosed(ctx) {
			panic("attempt to use a closed file")
		}

		err := fh.file.Close(ctx)
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

	ctx := v.Context()
	if fh.closed || fh.file.IsClosed(ctx) {
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
		outputVal := ioTable.Get(vm.NewString(ioFieldOutput))
		fh := getFileHandle(v, outputVal, "io.flush")
		ctx := v.Context()
		fh.checkOpen(ctx, "flush")

		if err := fh.file.Flush(ctx); err != nil {
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
			inputVal := ioTable.Get(vm.NewString(ioFieldInput))
			fh := getFileHandle(v, inputVal, "io.lines")
			ctx := v.Context()
			fh.checkOpen(ctx, "lines")
			f = fh.file
			toClose = false
		} else {
			arg := v.Get(1)
			if !arg.IsString() && !arg.IsNumber() {
				callerArgError(v, 1, "io.lines", fmt.Sprintf("string expected, got %s", v.ObjTypeName(arg)))
			}
			name := vm.ValueToString(arg)
			var err error
			ctx := v.Context()
			f, err = provider.Open(ctx, name, "r")
			if err != nil {
				_, errDesc := extractLuaFileError(err)
				panic(fmt.Sprintf("cannot open file '%s' (%s)", name, errDesc))
			}
			toClose = true
		}
		for i := 2; i <= v.ArgCount(); i++ {
			formats = append(formats, v.Get(i))
		}
		if len(formats) > maxArgLine {
			callerArgError(v, maxArgLine+2, "io.lines", "too many arguments")
		}

		closed := false
		// Return an iterator that reads lines
		v.Set(0, vm.NewNativeFunc(func(v *vm.VM) int {
			if closed {
				panic("file is already closed")
			}

			ctx := v.Context()
			if len(formats) == 0 {
				line, err := f.Read(ctx, "l")
				if err != nil {
					if err == io.EOF {
						if toClose {
							f.Close(ctx)
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

			results := doFileReadFormats(v, f, formats, 1)
			if results > 0 && v.Get(0).IsNil() {
				if toClose {
					f.Close(ctx)
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
			v.Set(3, makeFileHandle(v, f))
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
			v.Set(0, ioTable.Get(vm.NewString(ioFieldInput)))
			return 1
		}

		if arg.IsString() || arg.IsNumber() {
			// Open file as default input (coerce numbers to strings)
			fname := vm.ValueToString(arg)
			if arg.IsString() {
				fname = arg.AsString()
			}
			ctx := v.Context()
			f, err := provider.Open(ctx, fname, "r")
			if err != nil {
				_, errDesc := extractLuaFileError(err)
				panic(fmt.Sprintf("cannot open file '%s' (%s)", fname, errDesc))
			}
			handle := makeFileHandle(v, f)
			ioTable.SetString(ioFieldInput, handle)
			v.Set(0, handle)
			return 1
		}

		// Assume it's a file handle - set as default input
		fh := getFileHandle(v, arg, "io.input") // validate it's a file handle
		fh.checkOpen(v.Context(), "input")
		ioTable.SetString(ioFieldInput, arg)
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
			v.Set(0, ioTable.Get(vm.NewString(ioFieldOutput)))
			return 1
		}

		if arg.IsString() || arg.IsNumber() {
			// Open file as default output (coerce numbers to strings)
			fname := vm.ValueToString(arg)
			if arg.IsString() {
				fname = arg.AsString()
			}
			ctx := v.Context()
			f, err := provider.Open(ctx, fname, "w")
			if err != nil {
				_, errDesc := extractLuaFileError(err)
				panic(fmt.Sprintf("cannot open file '%s' (%s)", fname, errDesc))
			}
			handle := makeFileHandle(v, f)
			ioTable.SetString(ioFieldOutput, handle)
			v.Set(0, handle)
			return 1
		}

		// Assume it's a file handle - set as default output
		fh := getFileHandle(v, arg, "io.output") // validate it's a file handle
		fh.checkOpen(v.Context(), "output")
		ioTable.SetString(ioFieldOutput, arg)
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
		inputVal := ioTable.Get(vm.NewString(ioFieldInput))
		fh := getFileHandle(v, inputVal, "io.read")
		if fh.closed || fh.file.IsClosed(v.Context()) {
			panic("default input file is closed")
		}

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
		outputVal := ioTable.Get(vm.NewString(ioFieldOutput))
		fh := getFileHandle(v, outputVal, "io.write")
		if fh.closed || fh.file.IsClosed(v.Context()) {
			panic("default output file is closed")
		}

		return doFileWrite(v, fh.file, outputVal, 1)
	}
}

// fileWrite implements f:write(...) method.
func fileWrite(v *vm.VM) int {
	self := v.Get(1)
	fh := getFileHandle(v, self, "write")
	fh.checkOpen(v.Context(), "write")

	return doFileWrite(v, fh.file, self, 2)
}

// doFileWrite performs the actual write operation. firstArg is the index
// of the first data argument.
func doFileWrite(v *vm.VM, f vm.LuaFile, self vm.Value, firstArg int) int {
	ctx := v.Context()
	n := v.ArgCount() - (firstArg - 1)
	var totalBytes int64 // bytes written so far (for the 5.5 error counter)
	for i := firstArg; i < firstArg+n; i++ {
		arg := v.Get(i)
		var s string
		if arg.IsString() {
			s = arg.AsString()
		} else if arg.IsNumber() {
			// Lua 5.5: io.write uses the same float format as tostring().
			s = valueToString(arg)
		} else {
			fileArgError(v, i, "write", fmt.Sprintf("string expected, got %s", v.ObjTypeName(arg)))
		}
		err := f.Write(ctx, s)
		if err != nil {
			// Lua 5.5 g_write returns four values on a write error:
			// nil, error message, errno, and the number of bytes written
			// before the failure. golua writes whole arguments atomically, so
			// the failing write contributed 0 bytes and totalBytes is the sum
			// of the fully-written preceding arguments.
			errno, errDesc := extractLuaFileError(err)
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString(errDesc))
			v.Set(2, vm.NewInt(int64(errno)))
			v.Set(3, vm.NewInt(totalBytes))
			return 4
		}
		totalBytes += int64(len(s))
	}

	// Return the file handle (for chaining)
	v.Set(0, self)
	return 1
}

// fileClose implements f:close() method.
func fileClose(v *vm.VM) int {
	self := v.Get(1)
	fh := getFileHandle(v, self, "close")
	if fh.closeFn != nil {
		return fh.closeFn(v, fh)
	}

	ctx := v.Context()
	if fh.file.IsStd(ctx) {
		// Cannot close standard files
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString("cannot close standard file"))
		return 2
	}

	if fh.closed || fh.file.IsClosed(ctx) {
		panic("attempt to use a closed file")
	}

	err := fh.file.Close(ctx)
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
	fh.checkOpen(v.Context(), "lines")
	f := fh.file

	var formats []vm.Value
	for i := 2; i <= v.ArgCount(); i++ {
		formats = append(formats, v.Get(i))
	}
	if len(formats) > maxArgLine {
		callerArgError(v, maxArgLine+2, "lines", "too many arguments")
	}

	v.Set(0, vm.NewNativeFunc(func(v *vm.VM) int {
		ctx := v.Context()
		// Reference C Lua's io_readline iterator emits this specific
		// wording when the underlying file has been closed since the
		// iterator was created (liolib.c, "file is already closed").
		if fh.closed || fh.file.IsClosed(ctx) {
			panic("file is already closed")
		}

		if len(formats) == 0 {
			line, err := f.Read(ctx, "l")
			if err != nil {
				if err == io.EOF {
					return 0
				}
				panic(err.Error())
			}
			v.Set(0, vm.NewString(line))
			return 1
		}

		results := doFileReadFormats(v, f, formats, 2)
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
	fh.checkOpen(v.Context(), "seek")

	whence := seekCur
	if !v.Get(2).IsNil() {
		whence = v.Get(2).AsString()
	}

	// Validate whence before calling provider (invalid whence is a hard error)
	switch whence {
	case seekSet, seekCur, seekEnd:
		// valid
	default:
		fileArgError(v, 2, "seek", fmt.Sprintf("invalid option '%s'", whence))
	}

	var offset int64
	if !v.Get(3).IsNil() {
		arg := v.Get(3)
		var ok bool
		offset, ok = arg.ToInt()
		if !ok {
			// Distinguish "got non-number" from "got non-integer-representable
			// number" (fractional float, NaN, ±Inf, beyond int64), matching
			// reference Lua 5.4/5.5 wording.
			if arg.IsNumber() {
				fileArgError(v, 3, "seek", "number has no integer representation")
			}
			if _, isNum := arg.ToNumber(); isNum {
				fileArgError(v, 3, "seek", "number has no integer representation")
			}
			fileArgError(v, 3, "seek", fmt.Sprintf("number expected, got %s", v.ObjTypeName(arg)))
		}
	}

	ctx := v.Context()
	pos, err := fh.file.Seek(ctx, whence, offset)
	if err != nil {
		errno, errDesc := extractLuaFileError(err)
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString(errDesc))
		v.Set(2, vm.NewInt(int64(errno)))
		return 3
	}

	v.Set(0, vm.NewInt(pos))
	return 1
}

// fileSetVBuf implements f:setvbuf(mode [, size]) method.
func fileSetVBuf(v *vm.VM) int {
	self := v.Get(1)
	fh := getFileHandle(v, self, "setvbuf")
	fh.checkOpen(v.Context(), "setvbuf")

	mode := v.Get(2)
	if mode.IsNil() {
		fileArgError(v, 2, "setvbuf", "string expected, got nil")
	}
	var modeStr string
	if mode.IsString() {
		modeStr = mode.AsString()
	} else if mode.IsNumber() {
		modeStr = mode.String()
	} else {
		fileArgError(v, 2, "setvbuf", fmt.Sprintf("string expected, got %s", v.ObjTypeName(mode)))
	}

	var size int
	if !v.Get(3).IsNil() {
		arg := v.Get(3)
		sz, ok := arg.ToInt()
		if !ok {
			// Distinguish "got non-number" from "got non-integer-representable
			// number" (fractional float, NaN, ±Inf, beyond int64), matching
			// reference Lua 5.4/5.5 wording.
			if arg.IsNumber() {
				fileArgError(v, 3, "setvbuf", "number has no integer representation")
			}
			if _, isNum := arg.ToNumber(); isNum {
				fileArgError(v, 3, "setvbuf", "number has no integer representation")
			}
			fileArgError(v, 3, "setvbuf", fmt.Sprintf("number expected, got %s", v.ObjTypeName(arg)))
		}
		size = int(sz)
	}

	// Validate mode before calling provider (invalid mode is a hard error)
	switch modeStr {
	case vbufNo, vbufFull, vbufLine:
		// valid
	default:
		fileArgError(v, 2, "setvbuf", fmt.Sprintf("invalid option '%s'", modeStr))
	}

	ctx := v.Context()
	err := fh.file.SetVBuf(ctx, modeStr, size)
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
	ctx := v.Context()
	fh.checkOpen(ctx, "flush")

	err := fh.file.Flush(ctx)
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
	ctx := v.Context()
	if fh.closed || fh.file.IsClosed(ctx) {
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
	if v.ArgCount() < 1 {
		name, _ := v.ArgErrorFuncName()
		panic(fmt.Sprintf("bad argument #1 to '%s' (FILE* expected, got no value)", name))
	}
	if val.IsNil() {
		name, _ := v.ArgErrorFuncName()
		panic(fmt.Sprintf("bad argument #1 to '%s' (FILE* expected, got nil)", name))
	}
	ud := val.AsUserdata()
	if ud == nil {
		name, _ := v.ArgErrorFuncName()
		panic(fmt.Sprintf("bad argument #1 to '%s' (FILE* expected, got %s)", name, v.ObjTypeName(val)))
	}
	fh, ok := ud.Data.(*fileHandle)
	if !ok {
		name, _ := v.ArgErrorFuncName()
		panic(fmt.Sprintf("bad argument #1 to '%s' (FILE* expected, got %s)", name, v.ObjTypeName(val)))
	}
	if fh.gcCloseFn != nil {
		fh.gcCloseFn(fh)
		return 0
	}
	ctx := v.Context()
	if fh.closed || fh.file.IsClosed(ctx) || fh.file.IsStd(ctx) {
		return 0
	}
	fh.file.Close(ctx)
	fh.closed = true
	return 0
}
