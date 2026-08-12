package stdlib

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/iceisfun/golua/vm"
)

// This file implements io.popen: a file-like object backed by a spawned
// subprocess (LuaProcess). Split out of io.go.

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
		v.Set(0, makeFileHandleWithClose(v, pf, popenClose, popenCloseGC))
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
	// A pipe has no length to ask about, so the buffer grows with the data that
	// arrives. Sizing it from n instead would let f:read(1<<30) on a pipe that
	// yields two bytes allocate a gigabyte — a Go runtime OOM no pcall catches.
	return readPipeAtMost(f.reader, n)
}

// readPipeAtMost reads up to n bytes from an unbounded-length stream, doubling
// its buffer as data arrives instead of trusting the requested count.
func readPipeAtMost(r *bufio.Reader, n int) (string, error) {
	const initialRead = 64 << 10
	size := n
	if size > initialRead {
		size = initialRead
	}
	buf := make([]byte, 0, size)
	for len(buf) < n {
		if len(buf) == cap(buf) {
			grow := cap(buf) * 2
			if grow > n {
				grow = n
			}
			bigger := make([]byte, len(buf), grow)
			copy(bigger, buf)
			buf = bigger
		}
		read, err := r.Read(buf[len(buf):cap(buf)])
		buf = buf[:len(buf)+read]
		if err != nil {
			// Short reads are success: only a request that yielded nothing at
			// all reports the error (C's fread/feof behavior).
			if len(buf) == 0 {
				return "", err
			}
			break
		}
		if read == 0 {
			// A reader that yields neither bytes nor an error would spin here.
			break
		}
	}
	if len(buf) == 0 {
		return "", io.EOF
	}
	return string(buf), nil
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
