package vm

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// LuaProcessProvider is a capability interface for spawning and managing
// external processes. When provided to a VM, the exec module becomes available.
//
// This is distinct from LuaExecProvider (exec_provider.go), which provides
// synchronous shell execution for os.execute. LuaProcessProvider enables
// modern process control: streaming I/O, stdin interaction, and timed waits.
type LuaProcessProvider interface {
	// Spawn starts a new process. The context controls cancellation.
	// cmd is the executable path/name, args are command-line arguments.
	Spawn(ctx context.Context, cmd string, args []string, opts ProcessOptions) (LuaProcess, error)
}

// ProcessOptions configures how a process is spawned.
type ProcessOptions struct {
	Env    map[string]string // Environment variables (nil = inherit)
	Dir    string            // Working directory (empty = inherit)
	Stdin  bool              // Whether to create a stdin pipe
	Stdout bool              // Whether to capture stdout
	Stderr bool              // Whether to capture stderr
}

// LuaProcess represents a running or completed external process.
type LuaProcess interface {
	Read([]byte) (int, error)
	ReadLine() (string, error)
	Write([]byte) (int, error)
	CloseStdin() error
	Wait() ProcessResult
	WaitTimeout(timeout time.Duration) (ProcessResult, bool)
	IsComplete() bool
	Kill() error
	ReadStderr([]byte) (int, error)
	ReadStderrLine() (string, error)
}

// ProcessResult holds the outcome of a completed process.
type ProcessResult struct {
	Success bool
	Code    int
	Signal  int
}

// DefaultProcessProvider implements LuaProcessProvider using os/exec.
type DefaultProcessProvider struct{}

// NewDefaultProcessProvider creates a new DefaultProcessProvider.
func NewDefaultProcessProvider() *DefaultProcessProvider {
	return &DefaultProcessProvider{}
}

// Spawn starts a process using os/exec.
// Per Go docs, cmd.Wait() must not be called before all pipe reads complete
// (it closes the pipes). The returned process handles this by draining pipes
// in background goroutines and only calling cmd.Wait() after they finish.
func (p *DefaultProcessProvider) Spawn(ctx context.Context, cmd string, args []string, opts ProcessOptions) (LuaProcess, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	c := exec.CommandContext(ctx, cmd, args...)

	if opts.Dir != "" {
		c.Dir = opts.Dir
	}
	if opts.Env != nil {
		env := make([]string, 0, len(opts.Env))
		for k, v := range opts.Env {
			env = append(env, k+"="+v)
		}
		c.Env = env
	}

	proc := &defaultProcess{
		ctx:    ctx,
		doneCh: make(chan struct{}),
	}

	if opts.Stdin {
		stdin, err := c.StdinPipe()
		if err != nil {
			return nil, fmt.Errorf("stdin pipe: %w", err)
		}
		proc.stdin = stdin
	}

	// For stdout and stderr, we create pipes and read them in background
	// goroutines that feed data into thread-safe ring buffers. This ensures
	// the pipes are fully drained before cmd.Wait() is called.
	var pipeWg sync.WaitGroup

	if opts.Stdout {
		stdout, err := c.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("stdout pipe: %w", err)
		}
		proc.stdoutPipe = newPipeBuffer(stdout)
		pipeWg.Add(1)
		go func() {
			defer pipeWg.Done()
			proc.stdoutPipe.drain()
		}()
	}

	if opts.Stderr {
		stderr, err := c.StderrPipe()
		if err != nil {
			return nil, fmt.Errorf("stderr pipe: %w", err)
		}
		proc.stderrPipe = newPipeBuffer(stderr)
		pipeWg.Add(1)
		go func() {
			defer pipeWg.Done()
			proc.stderrPipe.drain()
		}()
	}

	if err := c.Start(); err != nil {
		return nil, err
	}

	proc.cmd = c

	// Wait for pipes to drain, then call cmd.Wait()
	go func() {
		pipeWg.Wait()
		err := c.Wait()
		proc.mu.Lock()
		proc.done = true
		proc.result = extractResult(err)
		proc.mu.Unlock()
		close(proc.doneCh)
	}()

	return proc, nil
}

// pipeBuffer drains a pipe reader into a buffer that can be read concurrently.
type pipeBuffer struct {
	reader io.ReadCloser
	mu     sync.Mutex
	buf    []byte
	pos    int // read position
	eof    bool
	cond   *sync.Cond
}

func newPipeBuffer(r io.ReadCloser) *pipeBuffer {
	pb := &pipeBuffer{reader: r}
	pb.cond = sync.NewCond(&pb.mu)
	return pb
}

// drain reads from the pipe until EOF, appending to buf.
func (pb *pipeBuffer) drain() {
	tmp := make([]byte, 4096)
	for {
		n, err := pb.reader.Read(tmp)
		if n > 0 {
			pb.mu.Lock()
			pb.buf = append(pb.buf, tmp[:n]...)
			pb.cond.Broadcast()
			pb.mu.Unlock()
		}
		if err != nil {
			pb.mu.Lock()
			pb.eof = true
			pb.cond.Broadcast()
			pb.mu.Unlock()
			return
		}
	}
}

// read copies available data into buf. Blocks until data is available or EOF.
func (pb *pipeBuffer) read(buf []byte) (int, error) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	for pb.pos >= len(pb.buf) && !pb.eof {
		pb.cond.Wait()
	}

	if pb.pos >= len(pb.buf) {
		return 0, io.EOF
	}

	n := copy(buf, pb.buf[pb.pos:])
	pb.pos += n
	return n, nil
}

// readLine reads until newline or EOF. Strips trailing \n and \r\n.
func (pb *pipeBuffer) readLine() (string, error) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	for {
		// Search for newline in available data
		start := pb.pos
		for i := pb.pos; i < len(pb.buf); i++ {
			if pb.buf[i] == '\n' {
				line := string(pb.buf[start:i])
				if len(line) > 0 && line[len(line)-1] == '\r' {
					line = line[:len(line)-1]
				}
				pb.pos = i + 1
				return line, nil
			}
		}

		if pb.eof {
			// Return remaining data without newline
			if pb.pos < len(pb.buf) {
				line := string(pb.buf[pb.pos:])
				pb.pos = len(pb.buf)
				return line, io.EOF
			}
			return "", io.EOF
		}

		pb.cond.Wait()
	}
}

func extractResult(err error) ProcessResult {
	if err == nil {
		return ProcessResult{Success: true, Code: 0}
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		status := exitErr.Sys().(syscall.WaitStatus)
		if status.Signaled() {
			return ProcessResult{
				Success: false,
				Code:    -1,
				Signal:  int(status.Signal()),
			}
		}
		return ProcessResult{
			Success: false,
			Code:    status.ExitStatus(),
		}
	}
	return ProcessResult{Success: false, Code: -1}
}

// defaultProcess implements LuaProcess backed by os/exec.
type defaultProcess struct {
	cmd        *exec.Cmd
	ctx        context.Context
	stdin      io.WriteCloser
	stdoutPipe *pipeBuffer
	stderrPipe *pipeBuffer
	doneCh     chan struct{}
	mu         sync.Mutex
	done       bool
	result     ProcessResult
}

func (p *defaultProcess) Read(buf []byte) (int, error) {
	if p.stdoutPipe == nil {
		return 0, io.EOF
	}
	return p.stdoutPipe.read(buf)
}

func (p *defaultProcess) ReadLine() (string, error) {
	if p.stdoutPipe == nil {
		return "", io.EOF
	}
	return p.stdoutPipe.readLine()
}

func (p *defaultProcess) Write(data []byte) (int, error) {
	if p.stdin == nil {
		return 0, fmt.Errorf("stdin not available")
	}
	return p.stdin.Write(data)
}

func (p *defaultProcess) CloseStdin() error {
	if p.stdin == nil {
		return fmt.Errorf("stdin not available")
	}
	return p.stdin.Close()
}

func (p *defaultProcess) Wait() ProcessResult {
	<-p.doneCh
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.result
}

func (p *defaultProcess) WaitTimeout(timeout time.Duration) (ProcessResult, bool) {
	select {
	case <-p.doneCh:
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.result, true
	case <-time.After(timeout):
		return ProcessResult{}, false
	}
}

func (p *defaultProcess) IsComplete() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.done
}

func (p *defaultProcess) Kill() error {
	if p.cmd.Process == nil {
		return fmt.Errorf("process not started")
	}
	return p.cmd.Process.Kill()
}

func (p *defaultProcess) ReadStderr(buf []byte) (int, error) {
	if p.stderrPipe == nil {
		return 0, io.EOF
	}
	return p.stderrPipe.read(buf)
}

func (p *defaultProcess) ReadStderrLine() (string, error) {
	if p.stderrPipe == nil {
		return "", io.EOF
	}
	return p.stderrPipe.readLine()
}
