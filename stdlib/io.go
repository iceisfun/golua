package stdlib

import (
	"bufio"
	"context"
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
	file      vm.LuaFile
	closed    bool
	closeFn   func(*vm.VM, *fileHandle) int
	gcCloseFn func(*fileHandle)
}

// makeFileHandle creates a file handle userdata wrapping a LuaFile.
func makeFileHandle(f vm.LuaFile) vm.Value {
	fh := &fileHandle{file: f}
	return vm.NewUserdataValue(fh, fileHandleMeta)
}

func makeFileHandleWithClose(f vm.LuaFile, closeFn func(*vm.VM, *fileHandle) int, gcCloseFn func(*fileHandle)) vm.Value {
	fh := &fileHandle{file: f, closeFn: closeFn, gcCloseFn: gcCloseFn}
	return vm.NewUserdataValue(fh, fileHandleMeta)
}

// fileArgError raises a "bad argument" error for file operations.
// idx is the 1-based argument position among explicit args (not counting self).
// firstArg is the stack position of the first explicit arg: 1 for io.* module
// functions (no self), 2 for f:method() calls (self at position 1).
// For method calls (firstArg==2), per Lua 5.4, self counts as arg #1
// and the function name shows as '?' since C can't resolve method names.
// For module calls (firstArg==1), the qualified name (e.g. "io.read") is used.
func fileArgError(idx int, name string, msg string, firstArg int) {
	if firstArg >= 2 {
		// Method call: offset arg number by 1 (self is arg #1), name is '?'
		panic(fmt.Sprintf("bad argument #%d to '?' (%s)", idx+1, msg))
	}
	// Module function call: use qualified name, no offset
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

type processReadCloser struct{ proc vm.LuaProcess }

func (p processReadCloser) Read(buf []byte) (int, error) { return p.proc.Read(buf) }

type popenFile struct {
	proc        vm.LuaProcess
	mode        string
	reader      *bufio.Reader
	closed      bool
	stdinClosed bool
	result      vm.ProcessResult
	waited      bool
}

func makeIoPopen(provider vm.LuaProcessProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		cmd := getString(v, 1, "io.popen")
		mode := "r"
		if !v.Get(2).IsNil() {
			mode = getString(v, 2, "io.popen")
		}
		if mode != "r" && mode != "w" && mode != "rb" && mode != "wb" {
			callerArgError(v, 2, "io.popen", "invalid mode")
		}

		ctx := v.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		opts := vm.ProcessOptions{}
		if mode[0] == 'r' {
			opts.Stdout = true
		} else {
			opts.Stdin = true
		}
		proc, err := provider.Spawn(ctx, "sh", []string{"-c", cmd}, opts)
		if err != nil {
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString(err.Error()))
			return 2
		}
		pf := &popenFile{proc: proc, mode: string(mode[0])}
		if pf.mode == "r" {
			pf.reader = bufio.NewReader(processReadCloser{proc: proc})
		}
		v.Set(0, makeFileHandleWithClose(pf, popenClose, popenCloseGC))
		return 1
	}
}

func (f *popenFile) Read(format string) (string, error) {
	if f.closed {
		return "", fmt.Errorf("attempt to use a closed file")
	}
	if f.mode != "r" {
		return "", fmt.Errorf("file not opened for reading")
	}
	if f.reader == nil {
		f.reader = bufio.NewReader(processReadCloser{proc: f.proc})
	}
	clean := strings.TrimPrefix(format, "*")
	switch clean {
	case "a":
		data, err := io.ReadAll(f.reader)
		if err != nil {
			return "", err
		}
		return string(data), nil
	case "l":
		return popenReadLine(f.reader, false)
	case "L":
		return popenReadLine(f.reader, true)
	case "n":
		return readNumberFromReader(f.reader)
	default:
		return "", fmt.Errorf("invalid read format: %s", format)
	}
}

func (f *popenFile) ReadBytes(n int) (string, error) {
	if f.closed {
		return "", fmt.Errorf("attempt to use a closed file")
	}
	if f.mode != "r" {
		return "", fmt.Errorf("file not opened for reading")
	}
	if n < 0 {
		return "", fmt.Errorf("not enough memory")
	}
	if f.reader == nil {
		f.reader = bufio.NewReader(processReadCloser{proc: f.proc})
	}
	if n == 0 {
		_, err := f.reader.Peek(1)
		if err != nil {
			return "", io.EOF
		}
		return "", nil
	}
	buf := make([]byte, n)
	read, err := io.ReadFull(f.reader, buf)
	if read == 0 && err != nil {
		return "", err
	}
	return string(buf[:read]), nil
}

func (f *popenFile) Write(data string) error {
	if f.closed {
		return fmt.Errorf("attempt to use a closed file")
	}
	if f.mode != "w" {
		return fmt.Errorf("file not opened for writing")
	}
	_, err := f.proc.Write([]byte(data))
	return err
}

func (f *popenFile) Seek(whence string, offset int64) (int64, error) {
	return 0, fmt.Errorf("seek not supported on popen file")
}

func (f *popenFile) Flush() error { return nil }

func (f *popenFile) SetVBuf(mode string, size int) error { return nil }

func (f *popenFile) Close() error {
	if f.closed {
		return fmt.Errorf("attempt to use a closed file")
	}
	if f.mode == "w" && !f.stdinClosed {
		_ = f.proc.CloseStdin()
		f.stdinClosed = true
	}
	if !f.waited {
		f.result = f.proc.Wait()
		f.waited = true
	}
	f.closed = true
	return nil
}

func (f *popenFile) IsClosed() bool { return f.closed }

func (f *popenFile) IsStd() bool { return false }

func popenClose(v *vm.VM, fh *fileHandle) int {
	if fh.closed || fh.file.IsClosed() {
		panic("attempt to use a closed file")
	}
	pf, ok := fh.file.(*popenFile)
	if !ok {
		panic("invalid popen file")
	}
	if err := pf.Close(); err != nil {
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString(err.Error()))
		return 2
	}
	fh.closed = true
	setExecResult(v, pf.result)
	return 3
}

func popenCloseGC(fh *fileHandle) {
	if fh.closed || fh.file.IsClosed() {
		return
	}
	_ = fh.file.Close()
	fh.closed = true
}

func setExecResult(v *vm.VM, result vm.ProcessResult) {
	if result.Signal != 0 {
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString("signal"))
		v.Set(2, vm.NewInt(int64(result.Signal)))
		return
	}
	if result.Success {
		v.Set(0, vm.True)
	} else {
		v.Set(0, vm.Nil)
	}
	v.Set(1, vm.NewString("exit"))
	v.Set(2, vm.NewInt(int64(result.Code)))
}

func popenReadLine(reader *bufio.Reader, keepNewline bool) (string, error) {
	var line strings.Builder
	for {
		b, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				if line.Len() > 0 {
					return line.String(), nil
				}
				return "", io.EOF
			}
			return "", err
		}
		if b == '\n' {
			if keepNewline {
				line.WriteByte('\n')
			}
			return line.String(), nil
		}
		line.WriteByte(b)
	}
}

func readNumberFromReader(reader *bufio.Reader) (string, error) {
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

	var buf []byte
	offset := 0
	for {
		peeked, _ := reader.Peek(offset + 1)
		if len(peeked) <= offset {
			break
		}
		b := peeked[offset]
		if isNumberChar(b) {
			buf = append(buf, b)
			offset++
		} else {
			break
		}
	}
	if len(buf) == 0 {
		return "", fmt.Errorf("not a number")
	}
	// Detect hex prefix: if buffer starts with 0x/0X, only accept hex parses.
	// Do not allow falling back to just "0" since the 0x signals hex intent.
	isHexPrefix := len(buf) >= 2 && buf[0] == '0' && (buf[1] == 'x' || buf[1] == 'X')
	minEnd := 1
	if isHexPrefix {
		minEnd = 3 // must have at least one hex digit after "0x"
	}
	bestEnd := 0
	for end := len(buf); end >= minEnd; end-- {
		s := string(buf[:end])
		if _, err := strconv.ParseInt(s, 0, 64); err == nil {
			bestEnd = end
			break
		}
		if _, err := strconv.ParseFloat(s, 64); err == nil {
			bestEnd = end
			break
		}
	}
	if bestEnd == 0 {
		return "", fmt.Errorf("not a number")
	}
	_, _ = reader.Discard(bestEnd)
	return string(buf[:bestEnd]), nil
}

func isNumberChar(b byte) bool {
	if b >= '0' && b <= '9' {
		return true
	}
	if b >= 'a' && b <= 'f' || b >= 'A' && b <= 'F' {
		return true
	}
	switch b {
	case '.', '-', '+', 'e', 'E', 'x', 'X', 'p', 'P':
		return true
	}
	return false
}

// makeIoOpen creates the io.open function.
func makeIoOpen(v *vm.VM, provider vm.LuaIoProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		name := v.Get(1)
		if name.IsNil() {
			callerArgError(v, 1, "io.open", "string expected, got nil")
		}
		var nameStr string
		if name.IsString() {
			nameStr = name.AsString()
		} else if name.IsNumber() {
			nameStr = vm.ValueToString(name)
		} else {
			callerArgError(v, 1, "io.open", fmt.Sprintf("string expected, got %s", name.Type()))
		}
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
func makeIoClose(ioTable *vm.Table) vm.NativeFunc {
	return func(v *vm.VM) int {
		val := v.Get(1)
		if val.IsNil() {
			val = ioTable.Get(vm.NewString("__output"))
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
			arg := v.Get(1)
			if !arg.IsString() && !arg.IsNumber() {
				callerArgError(v, 1, "io.lines", fmt.Sprintf("string expected, got %s", arg.Type()))
			}
			name := vm.ValueToString(arg)
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

			results := doFileReadFormats(v, f, formats, 1)
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

		if arg.IsString() || arg.IsNumber() {
			// Open file as default input (coerce numbers to strings)
			fname := vm.ValueToString(arg)
			if arg.IsString() {
				fname = arg.AsString()
			}
			f, err := provider.Open(fname, "r")
			if err != nil {
				_, errDesc := extractLuaFileError(err)
				panic(fmt.Sprintf("cannot open file '%s' (%s)", fname, errDesc))
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

		if arg.IsString() || arg.IsNumber() {
			// Open file as default output (coerce numbers to strings)
			fname := vm.ValueToString(arg)
			if arg.IsString() {
				fname = arg.AsString()
			}
			f, err := provider.Open(fname, "w")
			if err != nil {
				_, errDesc := extractLuaFileError(err)
				panic(fmt.Sprintf("cannot open file '%s' (%s)", fname, errDesc))
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
	return doFileReadFormats(v, f, formats, firstArg)
}

// doFileReadFormats performs the actual read operation using a slice of formats.
// firstArg controls error reporting: 1 for io.read (module func), 2 for f:read (method).
func doFileReadFormats(v *vm.VM, f vm.LuaFile, formats []vm.Value, firstArg int) int {
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
			count, ok := arg.ToInt()
			if !ok {
				fileArgError(results+1, "io.read", "number has no integer representation", firstArg)
			}
			if count < 0 {
				panic("not enough memory")
			}
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
				// Use results+1 as the user-visible argument index (1-based, for the format arg)
				fileArgError(results+1, "io.read", "invalid format", firstArg)
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
			fileArgError(results+1, "io.read", fmt.Sprintf("string expected, got %s", arg.Type()), firstArg)
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
			fileArgError(i-firstArg+1, "io.write", fmt.Sprintf("string expected, got %s", arg.Type()), firstArg)
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
	if fh.closeFn != nil {
		return fh.closeFn(v, fh)
	}

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
		fileArgError(1, "seek", fmt.Sprintf("invalid option '%s'", whence), 2)
	}

	var offset int64
	if !v.Get(3).IsNil() {
		var ok bool
		offset, ok = v.Get(3).ToInt()
		if !ok {
			fileArgError(2, "seek", "number expected", 2)
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
		fileArgError(1, "setvbuf", "string expected, got nil", 2)
	}
	modeStr := mode.AsString()

	var size int
	if !v.Get(3).IsNil() {
		sz, ok := v.Get(3).ToInt()
		if !ok {
			fileArgError(2, "setvbuf", "number expected", 2)
		}
		size = int(sz)
	}

	// Validate mode before calling provider (invalid mode is a hard error)
	switch modeStr {
	case "no", "full", "line":
		// valid
	default:
		fileArgError(1, "setvbuf", fmt.Sprintf("invalid option '%s'", modeStr), 2)
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
	if fh.gcCloseFn != nil {
		fh.gcCloseFn(fh)
		return 0
	}
	if fh.closed || fh.file.IsClosed() || fh.file.IsStd() {
		return 0
	}
	fh.file.Close()
	fh.closed = true
	return 0
}
