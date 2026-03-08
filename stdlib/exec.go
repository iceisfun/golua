package stdlib

import (
	"context"
	"fmt"
	"time"

	"github.com/iceisfun/golua/vm"
)

// processHandleMeta is a shared metatable for process handle userdata.
var processHandleMeta *vm.Table

// processMethodsTable contains methods available on process handles.
var processMethodsTable *vm.Table

func init() {
	processMethodsTable = vm.NewEmptyTable()
	processMethodsTable.SetString("read", vm.NewNativeFunc(processRead))
	processMethodsTable.SetString("readline", vm.NewNativeFunc(processReadLine))
	processMethodsTable.SetString("readlines", vm.NewNativeFunc(processReadLines))
	processMethodsTable.SetString("write", vm.NewNativeFunc(processWrite))
	processMethodsTable.SetString("close_stdin", vm.NewNativeFunc(processCloseStdin))
	processMethodsTable.SetString("wait", vm.NewNativeFunc(processWait))
	processMethodsTable.SetString("is_complete", vm.NewNativeFunc(processIsComplete))
	processMethodsTable.SetString("kill", vm.NewNativeFunc(processKill))
	processMethodsTable.SetString("exit_code", vm.NewNativeFunc(processExitCode))
	processMethodsTable.SetString("stderr", vm.NewNativeFunc(processStderr))

	processHandleMeta = vm.NewEmptyTable()
	processHandleMeta.SetString("__name", vm.NewString("PROCESS*"))
	processHandleMeta.SetString("__index", vm.NewTable(processMethodsTable))
	processHandleMeta.SetString("__tostring", vm.NewNativeFunc(processToString))
}

// processHandle is the Go data stored inside a process userdata value.
type processHandle struct {
	proc     vm.LuaProcess
	waited   bool
	result   vm.ProcessResult
	stdinOk  bool
	stdoutOk bool
	stderrOk bool
}

func makeProcessHandle(proc vm.LuaProcess, opts vm.ProcessOptions) vm.Value {
	ph := &processHandle{
		proc:     proc,
		stdinOk:  opts.Stdin,
		stdoutOk: opts.Stdout,
		stderrOk: opts.Stderr,
	}
	return vm.NewUserdataValue(ph, processHandleMeta)
}

func getProcessHandle(v *vm.VM, val vm.Value, funcName string) *processHandle {
	ud := val.AsUserdata()
	if ud == nil {
		panic(fmt.Sprintf("bad argument #1 to '%s' (PROCESS* expected, got %s)", funcName, v.ObjTypeName(val)))
	}
	ph, ok := ud.Data.(*processHandle)
	if !ok {
		panic(fmt.Sprintf("bad argument #1 to '%s' (PROCESS* expected)", funcName))
	}
	return ph
}

// openExec registers the exec module if a ProcessProvider is set.
func openExec(v *vm.VM) {
	provider := v.ProcessProvider()
	if provider == nil {
		return
	}

	execTable := vm.NewEmptyTable()
	execTable.SetString("run", vm.NewNativeFunc(makeExecRun(v, provider)))
	execTable.SetString("spawn", vm.NewNativeFunc(makeExecSpawn(v, provider)))
	execTable.SetString("run_shell", vm.NewNativeFunc(makeExecRunShell(v, provider)))

	v.SetGlobal("exec", vm.NewTable(execTable))
}

// extractExecOpts checks if the last argument is an options table.
// Returns the options table (or nil) and the adjusted argument count
// (excluding the options table from the string args).
func extractExecOpts(v *vm.VM, argc int) (vm.LuaTable, int) {
	if argc < 1 {
		return nil, argc
	}
	last := v.Get(argc)
	if last.IsTable() {
		return last.AsTable(), argc - 1
	}
	return nil, argc
}

// applyExecOpts reads Lua option fields from the options table into ProcessOptions.
func applyExecOpts(optsTable vm.LuaTable, opts *vm.ProcessOptions) {
	if optsTable == nil {
		return
	}
	mergeStderr := optsTable.Get(vm.NewString("merge_stderr"))
	if mergeStderr.IsBool() && mergeStderr.AsBool() {
		opts.MergeStderr = true
	}
}

// exec.run(cmd, arg1, arg2, ..., [opts]) -> {success, code, stdout, stderr}
func makeExecRun(luaVM *vm.VM, provider vm.LuaProcessProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		argc := v.ArgCount()
		if argc < 1 {
			panic("bad argument #1 to 'exec.run' (string expected, got no value)")
		}
		optsTable, argEnd := extractExecOpts(v, argc)
		cmd := checkString(v, 1, "exec.run")
		args := make([]string, 0, argEnd-1)
		for i := 2; i <= argEnd; i++ {
			args = append(args, checkString(v, i, "exec.run"))
		}

		ctx := v.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		opts := vm.ProcessOptions{
			Stdout: true,
			Stderr: true,
		}
		applyExecOpts(optsTable, &opts)
		if opts.MergeStderr {
			opts.Stderr = false
		}
		proc, err := provider.Spawn(ctx, cmd, args, opts)
		if err != nil {
			panic(fmt.Sprintf("exec.run: %s", err.Error()))
		}

		// Read all stdout (includes stderr when merged)
		stdout := readAll(proc)
		// Read all stderr (empty when merged)
		stderr := readAllStderr(proc)

		result := proc.Wait()

		// Build result table
		tbl := vm.NewEmptyTable()
		tbl.SetString("success", vm.NewBool(result.Success))
		tbl.SetString("code", vm.NewInt(int64(result.Code)))
		tbl.SetString("stdout", vm.NewString(stdout))
		tbl.SetString("stderr", vm.NewString(stderr))

		v.Set(0, vm.NewTable(tbl))
		return 1
	}
}

// exec.spawn(cmd, arg1, arg2, ..., [opts]) -> process
func makeExecSpawn(luaVM *vm.VM, provider vm.LuaProcessProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		argc := v.ArgCount()
		if argc < 1 {
			panic("bad argument #1 to 'exec.spawn' (string expected, got no value)")
		}
		optsTable, argEnd := extractExecOpts(v, argc)
		cmd := checkString(v, 1, "exec.spawn")
		args := make([]string, 0, argEnd-1)
		for i := 2; i <= argEnd; i++ {
			args = append(args, checkString(v, i, "exec.spawn"))
		}

		ctx := v.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		opts := vm.ProcessOptions{
			Stdin:  true,
			Stdout: true,
			Stderr: true,
		}
		applyExecOpts(optsTable, &opts)
		if opts.MergeStderr {
			opts.Stderr = false
		}
		proc, err := provider.Spawn(ctx, cmd, args, opts)
		if err != nil {
			panic(fmt.Sprintf("exec.spawn: %s", err.Error()))
		}

		v.Set(0, makeProcessHandle(proc, opts))
		return 1
	}
}

// exec.run_shell(cmdline, [opts]) -> {success, code, stdout, stderr}
func makeExecRunShell(luaVM *vm.VM, provider vm.LuaProcessProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		if v.ArgCount() < 1 {
			panic("bad argument #1 to 'exec.run_shell' (string expected, got no value)")
		}
		cmdline := checkString(v, 1, "exec.run_shell")

		// Check for options table as second argument
		var optsTable vm.LuaTable
		if v.ArgCount() >= 2 && v.Get(2).IsTable() {
			optsTable = v.Get(2).AsTable()
		}

		ctx := v.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		opts := vm.ProcessOptions{
			Stdout: true,
			Stderr: true,
		}
		applyExecOpts(optsTable, &opts)
		if opts.MergeStderr {
			opts.Stderr = false
		}
		proc, err := provider.Spawn(ctx, "sh", []string{"-c", cmdline}, opts)
		if err != nil {
			panic(fmt.Sprintf("exec.run_shell: %s", err.Error()))
		}

		stdout := readAll(proc)
		stderr := readAllStderr(proc)
		result := proc.Wait()

		tbl := vm.NewEmptyTable()
		tbl.SetString("success", vm.NewBool(result.Success))
		tbl.SetString("code", vm.NewInt(int64(result.Code)))
		tbl.SetString("stdout", vm.NewString(stdout))
		tbl.SetString("stderr", vm.NewString(stderr))

		v.Set(0, vm.NewTable(tbl))
		return 1
	}
}

// Process methods

// p:read() -> string or nil
func processRead(v *vm.VM) int {
	ph := getProcessHandle(v, v.Get(1), "process:read")
	buf := make([]byte, 4096)
	n, err := ph.proc.Read(buf)
	if err != nil || n == 0 {
		v.Set(0, vm.Nil)
		return 1
	}
	v.Set(0, vm.NewString(string(buf[:n])))
	return 1
}

// p:readline() -> string or nil
func processReadLine(v *vm.VM) int {
	ph := getProcessHandle(v, v.Get(1), "process:readline")
	line, err := ph.proc.ReadLine()
	if err != nil && len(line) == 0 {
		v.Set(0, vm.Nil)
		return 1
	}
	v.Set(0, vm.NewString(line))
	return 1
}

// p:readlines() -> iterator function
func processReadLines(v *vm.VM) int {
	ph := getProcessHandle(v, v.Get(1), "process:readlines")
	iter := func(v *vm.VM) int {
		line, err := ph.proc.ReadLine()
		if err != nil && len(line) == 0 {
			v.Set(0, vm.Nil)
			return 1
		}
		v.Set(0, vm.NewString(line))
		return 1
	}
	v.Set(0, vm.NewNativeFunc(iter))
	return 1
}

// p:write(data) -> process (for chaining)
func processWrite(v *vm.VM) int {
	ph := getProcessHandle(v, v.Get(1), "process:write")
	data := checkString(v, 2, "process:write")
	_, err := ph.proc.Write([]byte(data))
	if err != nil {
		panic(fmt.Sprintf("process:write: %s", err.Error()))
	}
	v.Set(0, v.Get(1)) // return self for chaining
	return 1
}

// p:close_stdin()
func processCloseStdin(v *vm.VM) int {
	ph := getProcessHandle(v, v.Get(1), "process:close_stdin")
	if err := ph.proc.CloseStdin(); err != nil {
		panic(fmt.Sprintf("process:close_stdin: %s", err.Error()))
	}
	return 0
}

// p:wait(ms?) -> result table [, done boolean]
func processWait(v *vm.VM) int {
	ph := getProcessHandle(v, v.Get(1), "process:wait")

	// Check for timeout argument
	timeoutArg := v.Get(2)
	if !timeoutArg.IsNil() {
		ms, ok := timeoutArg.ToInt()
		if !ok {
			panic("bad argument #2 to 'process:wait' (number expected)")
		}
		result, done := ph.proc.WaitTimeout(time.Duration(ms) * time.Millisecond)
		if !done {
			v.Set(0, vm.Nil)
			v.Set(1, vm.False)
			return 2
		}
		ph.waited = true
		ph.result = result
		v.Set(0, makeResultTable(result))
		v.Set(1, vm.True)
		return 2
	}

	result := ph.proc.Wait()
	ph.waited = true
	ph.result = result
	v.Set(0, makeResultTable(result))
	return 1
}

// p:is_complete() -> boolean
func processIsComplete(v *vm.VM) int {
	ph := getProcessHandle(v, v.Get(1), "process:is_complete")
	v.Set(0, vm.NewBool(ph.proc.IsComplete()))
	return 1
}

// p:kill()
func processKill(v *vm.VM) int {
	ph := getProcessHandle(v, v.Get(1), "process:kill")
	if err := ph.proc.Kill(); err != nil {
		panic(fmt.Sprintf("process:kill: %s", err.Error()))
	}
	return 0
}

// p:exit_code() -> number or nil
func processExitCode(v *vm.VM) int {
	ph := getProcessHandle(v, v.Get(1), "process:exit_code")
	if !ph.proc.IsComplete() {
		v.Set(0, vm.Nil)
		return 1
	}
	if !ph.waited {
		result := ph.proc.Wait()
		ph.waited = true
		ph.result = result
	}
	v.Set(0, vm.NewInt(int64(ph.result.Code)))
	return 1
}

// p:stderr() -> string or nil
func processStderr(v *vm.VM) int {
	ph := getProcessHandle(v, v.Get(1), "process:stderr")
	line, err := ph.proc.ReadStderrLine()
	if err != nil && len(line) == 0 {
		v.Set(0, vm.Nil)
		return 1
	}
	v.Set(0, vm.NewString(line))
	return 1
}

func processToString(v *vm.VM) int {
	ud := v.Get(1).AsUserdata()
	if ud == nil {
		v.Set(0, vm.NewString("process (invalid)"))
		return 1
	}
	v.Set(0, vm.NewString(fmt.Sprintf("process: %p", ud)))
	return 1
}

// helpers

func checkString(v *vm.VM, idx int, funcName string) string {
	val := v.Get(idx)
	if val.IsString() {
		return val.AsString()
	}
	// Allow number coercion
	if val.IsNumber() {
		return vm.ValueToString(val)
	}
	panic(fmt.Sprintf("bad argument #%d to '%s' (string expected, got %s)", idx, funcName, v.ObjTypeName(val)))
}

func makeResultTable(result vm.ProcessResult) vm.Value {
	tbl := vm.NewEmptyTable()
	tbl.SetString("success", vm.NewBool(result.Success))
	tbl.SetString("code", vm.NewInt(int64(result.Code)))
	if result.Signal > 0 {
		tbl.SetString("signal", vm.NewInt(int64(result.Signal)))
	}
	return vm.NewTable(tbl)
}

func readAll(proc vm.LuaProcess) string {
	buf := make([]byte, 4096)
	var data []byte
	for {
		n, err := proc.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	return string(data)
}

func readAllStderr(proc vm.LuaProcess) string {
	buf := make([]byte, 4096)
	var data []byte
	for {
		n, err := proc.ReadStderr(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	return string(data)
}
