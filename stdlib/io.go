package stdlib

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/iceisfun/golua/vm"
)

// maxArgLine is the maximum number of format arguments for f:lines()/io.lines().
// Matches Lua 5.4's MAXARGLINE (liolib.c).
const maxArgLine = 250

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
	return vm.NewUserdataValueUV(fh, fileHandleMeta, 0)
}

func makeFileHandleWithClose(f vm.LuaFile, closeFn func(*vm.VM, *fileHandle) int, gcCloseFn func(*fileHandle)) vm.Value {
	fh := &fileHandle{file: f, closeFn: closeFn, gcCloseFn: gcCloseFn}
	return vm.NewUserdataValueUV(fh, fileHandleMeta, 0)
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
		stdinHandle = makeFileHandle(f)
		ioTable.SetString("stdin", stdinHandle)
	}
	if f := provider.Stdout(ctx); f != nil {
		stdoutHandle = makeFileHandle(f)
		ioTable.SetString("stdout", stdoutHandle)
	}
	if f := provider.Stderr(ctx); f != nil {
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

func (f *popenFile) Read(ctx context.Context, format string) (string, error) {
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

func (f *popenFile) ReadBytes(ctx context.Context, n int) (string, error) {
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

func (f *popenFile) Write(ctx context.Context, data string) error {
	if f.closed {
		return fmt.Errorf("attempt to use a closed file")
	}
	if f.mode != "w" {
		return fmt.Errorf("file not opened for writing")
	}
	_, err := f.proc.Write([]byte(data))
	return err
}

func (f *popenFile) Seek(ctx context.Context, whence string, offset int64) (int64, error) {
	return 0, fmt.Errorf("seek not supported on popen file")
}

func (f *popenFile) Flush(ctx context.Context) error { return nil }

func (f *popenFile) SetVBuf(ctx context.Context, mode string, size int) error { return nil }

func (f *popenFile) Close(ctx context.Context) error {
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

func (f *popenFile) IsClosed(ctx context.Context) bool { return f.closed }

func (f *popenFile) IsStd(ctx context.Context) bool { return false }

func popenClose(v *vm.VM, fh *fileHandle) int {
	ctx := v.Context()
	if fh.closed || fh.file.IsClosed(ctx) {
		panic("attempt to use a closed file")
	}
	pf, ok := fh.file.(*popenFile)
	if !ok {
		panic("invalid popen file")
	}
	if err := pf.Close(ctx); err != nil {
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString(err.Error()))
		return 2
	}
	fh.closed = true
	setExecResult(v, pf.result)
	return 3
}

func popenCloseGC(fh *fileHandle) {
	ctx := context.Background()
	if fh.closed || fh.file.IsClosed(ctx) {
		return
	}
	_ = fh.file.Close(ctx)
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

		ctx := v.Context()
		f, err := provider.Open(ctx, nameStr, mode)
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
		v.Set(0, makeFileHandle(f))
		return 1
	}
}

// makeIoClose creates the io.close([file]) function.
func makeIoClose(ioTable *vm.Table) vm.NativeFunc {
	return func(v *vm.VM) int {
		val := v.Get(1)
		if v.ArgCount() == 0 {
			// io.close() with no args: close default output
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
		outputVal := ioTable.Get(vm.NewString("__output"))
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
			inputVal := ioTable.Get(vm.NewString("__input"))
			fh := getFileHandle(v, inputVal, "io.lines")
			ctx := v.Context()
			fh.checkOpen(ctx, "lines")
			f = fh.file
			toClose = false
		} else {
			arg := v.Get(1)
			if !arg.IsString() && !arg.IsNumber() {
				callerArgError(v, 1, "io.lines", fmt.Sprintf("string expected, got %s", arg.Type()))
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
			ctx := v.Context()
			f, err := provider.Open(ctx, fname, "r")
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
		fh := getFileHandle(v, arg, "io.input") // validate it's a file handle
		fh.checkOpen(v.Context(), "input")
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
			ctx := v.Context()
			f, err := provider.Open(ctx, fname, "w")
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
		fh := getFileHandle(v, arg, "io.output") // validate it's a file handle
		fh.checkOpen(v.Context(), "output")
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
		outputVal := ioTable.Get(vm.NewString("__output"))
		fh := getFileHandle(v, outputVal, "io.write")
		if fh.closed || fh.file.IsClosed(v.Context()) {
			panic("default output file is closed")
		}

		return doFileWrite(v, fh.file, outputVal, 1)
	}
}

// fileRead implements f:read(...) method.
func fileRead(v *vm.VM) int {
	fh := getFileHandle(v, v.Get(1), "read")
	fh.checkOpen(v.Context(), "read")

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
	ctx := v.Context()
	if len(formats) == 0 {
		// Default: read a line
		line, err := f.Read(ctx, "l")
		if err != nil {
			if err == io.EOF {
				v.Set(0, vm.Nil)
				return 1
			}
			errno, errDesc := extractLuaFileError(err)
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString(errDesc))
			v.Set(2, vm.NewInt(int64(errno)))
			return 3
		}
		v.Set(0, vm.NewString(line))
		return 1
	}

	// Process each format argument.
	// Track OS-level I/O errors: Lua 5.4 checks ferror(f) after the loop and
	// returns nil, errmsg, errno if the underlying file had an error.
	// Normal failures (EOF, no-number-found) produce nil and stop further reads
	// (Lua 5.4 uses a success flag that gates the loop).
	n := len(formats)
	v.EnsureStack(v.Base() + n)
	results := 0
	var fileErr error
	success := true
	for _, arg := range formats {
		if !success {
			v.Set(results, vm.Nil)
			results++
			continue
		}
		if arg.IsNumber() {
			// Read N bytes
			count, ok := arg.ToInt()
			if !ok {
				fileArgError(v, firstArg+results, "read", "number has no integer representation")
			}
			if count < 0 {
				panic("not enough memory")
			}
			if count == 0 {
				// Read 0 bytes: test if at EOF
				data, err := f.ReadBytes(ctx, 0)
				if err != nil {
					if isFileError(err) {
						fileErr = err
					}
					v.Set(results, vm.Nil)
					success = false
				} else {
					v.Set(results, vm.NewString(data))
				}
			} else {
				data, err := f.ReadBytes(ctx, int(count))
				if err != nil {
					if isFileError(err) {
						fileErr = err
					}
					v.Set(results, vm.Nil)
					success = false
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
				fileArgError(v, firstArg+results, "read", "invalid format")
			}
			data, err := f.Read(ctx, format)
			if err != nil {
				if isFileError(err) {
					fileErr = err
				}
				v.Set(results, vm.Nil)
				success = false
			} else {
				// Check if format is "n" or "*n" for number parsing
				cleanFmt := strings.TrimPrefix(format, "*")
				if len(cleanFmt) > 0 && cleanFmt[0] == 'n' {
					// Parse as number matching Lua 5.4 semantics:
					// - Leading zeros are decimal (not octal)
					// - Hex floats (0x1.8, 0xABp0) produce floats
					// - Overflow to ±Inf is valid
					v.Set(results, parseReadNumber(data))
				} else {
					v.Set(results, vm.NewString(data))
				}
			}
		} else {
			// Invalid format type
			fileArgError(v, firstArg+results, "read", fmt.Sprintf("string expected, got %s", arg.Type()))
		}
		results++
	}
	// Lua 5.4: if the file had an OS-level I/O error (ferror), return nil, msg, errno.
	if fileErr != nil {
		errno, errDesc := extractLuaFileError(fileErr)
		v.Set(0, vm.Nil)
		v.Set(1, vm.NewString(errDesc))
		v.Set(2, vm.NewInt(int64(errno)))
		return 3
	}
	return results
}

// parseReadNumber converts a string read by read("n") to a Lua number value.
// Matches Lua 5.4 semantics: leading zeros are decimal (not octal), hex floats
// (0x1.8, 0xABp0) produce floats, and overflow to ±Inf is valid.
func parseReadNumber(data string) vm.Value {
	if len(data) == 0 {
		return vm.Nil
	}

	// Determine sign and hex prefix, accounting for optional leading sign.
	digitStart := 0
	if len(data) > 0 && (data[0] == '+' || data[0] == '-') {
		digitStart = 1
	}
	isHex := len(data) > digitStart+2 && data[digitStart] == '0' && (data[digitStart+1] == 'x' || data[digitStart+1] == 'X')
	hexBody := ""
	if isHex && len(data) > digitStart+2 {
		hexBody = data[digitStart+2:]
	}
	isHexFloat := isHex && strings.ContainsAny(hexBody, ".pP")
	// For hex numbers, only '.', 'p', 'P' indicate float (not 'e'/'E' which are hex digits).
	hasFloatIndicator := !isHex && strings.ContainsAny(data, ".eE")

	// Hex floats always produce float type
	if isHexFloat {
		fv, err := strconv.ParseFloat(data, 64)
		if err != nil {
			// Go doesn't support hex floats without p exponent (e.g. "0x1.8").
			// Parse manually: integer part + fractional part.
			fv, ok := parseHexFloat(data)
			if !ok {
				return vm.Nil
			}
			return vm.NewFloat(fv)
		}
		return vm.NewFloat(fv)
	}

	// Try integer first (base 10 for decimal, base 0 for hex with prefix)
	if !hasFloatIndicator {
		if isHex {
			// Use base 0 which handles [+-]0x prefix natively
			if iv, err := strconv.ParseInt(data, 0, 64); err == nil {
				return vm.NewInt(iv)
			}
			// Fallback: try unsigned parse for values > max int64 (e.g. 0x8000000000000001)
			// then wrap to int64, matching Lua 5.4's lua_stringtonumber behavior.
			sign := int64(1)
			hexStr := hexBody
			if digitStart > 0 && data[0] == '-' {
				sign = -1
			}
			if u, err := strconv.ParseUint(hexStr, 16, 64); err == nil {
				return vm.NewInt(sign * int64(u))
			}
		} else {
			if iv, err := strconv.ParseInt(data, 10, 64); err == nil {
				return vm.NewInt(iv)
			}
		}
	}

	// Try float
	fv, err := strconv.ParseFloat(data, 64)
	if err != nil {
		// Accept overflow results (ErrRange) — produces ±Inf
		if numErr, ok := err.(*strconv.NumError); ok && numErr.Err == strconv.ErrRange {
			if math.IsInf(fv, 0) {
				return vm.NewFloat(fv)
			}
		}
		return vm.Nil
	}

	return vm.NewFloat(fv)
}

// parseHexFloat parses hex floats without p exponent (e.g. "0x1.8" = 1.5).
// Go's strconv.ParseFloat requires p exponent for hex floats.
func parseHexFloat(data string) (float64, bool) {
	s := data
	neg := false
	if len(s) > 0 && (s[0] == '+' || s[0] == '-') {
		neg = s[0] == '-'
		s = s[1:]
	}
	if len(s) < 3 || s[0] != '0' || (s[1] != 'x' && s[1] != 'X') {
		return 0, false
	}
	s = s[2:]

	// Split at dot
	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]
	fracPart := ""
	if len(parts) > 1 {
		fracPart = "" // will be set below
		rest := parts[1]
		// Check for p/P exponent
		pIdx := strings.IndexAny(rest, "pP")
		if pIdx >= 0 {
			// Has exponent — let Go handle it by appending p0
			withP := data
			if !strings.ContainsAny(data, "pP") {
				withP = data + "p0"
			}
			fv, err := strconv.ParseFloat(withP, 64)
			return fv, err == nil
		}
		fracPart = rest
	}

	// Parse integer part
	var result float64
	if intPart != "" {
		iv, err := strconv.ParseUint(intPart, 16, 64)
		if err != nil {
			return 0, false
		}
		result = float64(iv)
	}

	// Parse fractional part
	if fracPart != "" {
		frac := 0.0
		scale := 1.0 / 16.0
		for _, c := range fracPart {
			var d int
			switch {
			case c >= '0' && c <= '9':
				d = int(c - '0')
			case c >= 'a' && c <= 'f':
				d = int(c-'a') + 10
			case c >= 'A' && c <= 'F':
				d = int(c-'A') + 10
			default:
				return 0, false
			}
			frac += float64(d) * scale
			scale /= 16.0
		}
		result += frac
	}

	if neg {
		result = -result
	}
	return result, true
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
			fileArgError(v, i, "write", fmt.Sprintf("string expected, got %s", arg.Type()))
		}
		err := f.Write(ctx, s)
		if err != nil {
			errno, errDesc := extractLuaFileError(err)
			v.Set(0, vm.Nil)
			v.Set(1, vm.NewString(errDesc))
			v.Set(2, vm.NewInt(int64(errno)))
			return 3
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
		fh.checkOpen(ctx, "lines iterator")

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

	whence := "cur"
	if !v.Get(2).IsNil() {
		whence = v.Get(2).AsString()
	}

	// Validate whence before calling provider (invalid whence is a hard error)
	switch whence {
	case "set", "cur", "end":
		// valid
	default:
		fileArgError(v, 2, "seek", fmt.Sprintf("invalid option '%s'", whence))
	}

	var offset int64
	if !v.Get(3).IsNil() {
		var ok bool
		offset, ok = v.Get(3).ToInt()
		if !ok {
			fileArgError(v, 3, "seek", "number expected")
		}
	}

	ctx := v.Context()
	pos, err := fh.file.Seek(ctx, whence, offset)
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
		fileArgError(v, 2, "setvbuf", fmt.Sprintf("string expected, got %s", mode.Type()))
	}

	var size int
	if !v.Get(3).IsNil() {
		sz, ok := v.Get(3).ToInt()
		if !ok {
			fileArgError(v, 3, "setvbuf", "number expected")
		}
		size = int(sz)
	}

	// Validate mode before calling provider (invalid mode is a hard error)
	switch modeStr {
	case "no", "full", "line":
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
