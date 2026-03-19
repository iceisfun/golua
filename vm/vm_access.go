package vm

import "context"

// Stack access for native functions

// Base returns the base stack index for the current call.
func (vm *VM) Base() int {
	if len(vm.callStack) == 0 {
		return 0
	}
	return vm.callStack[len(vm.callStack)-1].base
}

// Top returns the top of the stack.
func (vm *VM) Top() int {
	return vm.top
}

// Get returns the value at the given stack index (relative to base).
func (vm *VM) Get(idx int) Value {
	base := vm.Base()
	if idx >= 0 {
		return vm.stack[base+idx]
	}
	// Negative index counts from top
	return vm.stack[vm.top+idx]
}

// Set sets the value at the given stack index (relative to base).
func (vm *VM) Set(idx int, v Value) {
	base := vm.Base()
	if idx >= 0 {
		vm.stack[base+idx] = v
	} else {
		vm.stack[vm.top+idx] = v
	}
}

// ArgCount returns the number of arguments passed to the current function.
func (vm *VM) ArgCount() int {
	if len(vm.callStack) == 0 {
		return 0
	}
	frame := &vm.callStack[len(vm.callStack)-1]
	if frame.argc != UseVMTop {
		return frame.argc
	}
	return vm.top - frame.base - 1
}

// EnterNonYieldable marks a native callback region where coroutine.yield()
// must fail with "attempt to yield across a C-call boundary".
// The returned function must be called (typically via defer) to restore state.
func (vm *VM) EnterNonYieldable() func() {
	vm.nonYieldableDepth++
	return func() {
		if vm.nonYieldableDepth <= 0 {
			panic("internal error: non-yieldable depth underflow")
		}
		vm.nonYieldableDepth--
	}
}

// IsYieldableContext reports whether the current execution context can yield.
// Main thread (no coroutine channels) is never yieldable.
func (vm *VM) IsYieldableContext() bool {
	return vm.yieldCh != nil && vm.nonYieldableDepth == 0
}

// EnterUserProtected marks execution as running under pcall/xpcall.
// The returned function must be called to restore state.
func (vm *VM) EnterUserProtected() func() {
	vm.userProtectedBases = append(vm.userProtectedBases, len(vm.callStack))
	return func() {
		n := len(vm.userProtectedBases)
		if n == 0 {
			panic("internal error: user-protected depth underflow")
		}
		vm.userProtectedBases = vm.userProtectedBases[:n-1]
	}
}

// InUserProtected reports whether execution is currently under pcall/xpcall.
func (vm *VM) InUserProtected() bool {
	return len(vm.userProtectedBases) > 0
}

// InUserProtectedDirectCallee reports whether the currently executing function
// is the direct callee of the innermost active pcall/xpcall.
func (vm *VM) InUserProtectedDirectCallee() bool {
	n := len(vm.userProtectedBases)
	if n == 0 {
		return false
	}
	base := vm.userProtectedBases[n-1]
	return len(vm.callStack) == base+1
}

// EnterDirectProtectedLoad marks a pcall/xpcall invocation that directly
// targets load() as its callee.
func (vm *VM) EnterDirectProtectedLoad() func() {
	vm.directProtectedLoadDepth++
	return func() {
		if vm.directProtectedLoadDepth <= 0 {
			panic("internal error: direct-protected-load depth underflow")
		}
		vm.directProtectedLoadDepth--
	}
}

// InDirectProtectedLoad reports whether load() is currently being called as
// the direct function argument of pcall/xpcall.
func (vm *VM) InDirectProtectedLoad() bool {
	return vm.directProtectedLoadDepth > 0
}

// Push pushes a value onto the stack.
// Must be called from within a ProtectedCall boundary (i.e., from a
// NativeFunc). Panics on stack overflow, caught by ProtectedCall's recover.
func (vm *VM) Push(v Value) {
	vm.ensureStack(vm.top + 1)
	vm.stack[vm.top] = v
	vm.top++
}

// Pop pops a value from the stack.
func (vm *VM) Pop() Value {
	vm.top--
	return vm.stack[vm.top]
}

// Provider getters/setters

// SetCodeProvider sets the code provider for this VM.
// The code provider is used by load, loadfile, and dofile to resolve and load Lua chunks.
func (vm *VM) SetCodeProvider(provider LuaCodeProvider) {
	vm.codeProvider = provider
	vm.registerProvider(provider)
}

// CodeProvider returns the current code provider, or nil if none is set.
func (vm *VM) CodeProvider() LuaCodeProvider {
	return vm.codeProvider
}

// SetVMID sets the VM identifier (used in caller context for code loading).
func (vm *VM) SetVMID(id string) {
	vm.vmID = id
}

// VMID returns the VM identifier.
func (vm *VM) VMID() string {
	return vm.vmID
}

// SetChunkName sets the name of the currently executing chunk.
func (vm *VM) SetChunkName(name string) {
	vm.chunkName = name
}

// ChunkName returns the name of the currently executing chunk.
func (vm *VM) ChunkName() string {
	return vm.chunkName
}

// CallerContext returns the current caller context for code loading.
func (vm *VM) CallerContext() *LuaCallerContext {
	return &LuaCallerContext{
		ScriptName: vm.chunkName,
		VMID:       vm.vmID,
		CallDepth:  len(vm.callStack),
	}
}

// SetIoProvider sets the IO provider for this VM.
func (vm *VM) SetIoProvider(provider LuaIoProvider) {
	vm.ioProvider = provider
	vm.registerProvider(provider)
}

// IoProvider returns the current IO provider, or nil if none is set.
func (vm *VM) IoProvider() LuaIoProvider {
	return vm.ioProvider
}

// SetOsProvider sets the OS provider for this VM.
func (vm *VM) SetOsProvider(provider LuaOsProvider) {
	vm.osProvider = provider
	vm.registerProvider(provider)
}

// OsProvider returns the current OS provider, or nil if none is set.
func (vm *VM) OsProvider() LuaOsProvider {
	return vm.osProvider
}

// SetExecProvider sets the exec provider for this VM.
func (vm *VM) SetExecProvider(provider LuaExecProvider) {
	vm.execProvider = provider
	vm.registerProvider(provider)
}

// ExecProvider returns the current exec provider, or nil if none is set.
func (vm *VM) ExecProvider() LuaExecProvider {
	return vm.execProvider
}

// SetExitHandler sets the exit handler for this VM.
func (vm *VM) SetExitHandler(handler LuaExitHandler) {
	vm.exitHandler = handler
	vm.registerProvider(handler)
}

// ExitHandler returns the current exit handler, or nil if none is set.
func (vm *VM) ExitHandler() LuaExitHandler {
	return vm.exitHandler
}

// SetDebugProvider sets the debug provider for this VM.
func (vm *VM) SetDebugProvider(provider LuaDebugProvider) {
	vm.debugProvider = provider
	vm.registerProvider(provider)
}

// DebugProvider returns the current debug provider, or nil if none is set.
func (vm *VM) DebugProvider() LuaDebugProvider {
	return vm.debugProvider
}

// SetChanProvider sets the channel provider for this VM.
func (vm *VM) SetChanProvider(provider LuaChanProvider) {
	vm.chanProvider = provider
	vm.registerProvider(provider)
}

// ChanProvider returns the current channel provider, or nil if none is set.
func (vm *VM) ChanProvider() LuaChanProvider {
	return vm.chanProvider
}

// SetTimeProvider sets the time provider for this VM.
func (vm *VM) SetTimeProvider(provider LuaTimeProvider) {
	vm.timeProvider = provider
	vm.registerProvider(provider)
}

// TimeProvider returns the current time provider, or nil if none is set.
func (vm *VM) TimeProvider() LuaTimeProvider {
	return vm.timeProvider
}

// SetLoadLibProvider sets the package.loadlib provider for this VM.
func (vm *VM) SetLoadLibProvider(provider LuaLoadLibProvider) {
	vm.loadLibProvider = provider
	vm.registerProvider(provider)
}

// LoadLibProvider returns the current package.loadlib provider, or nil if none is set.
func (vm *VM) LoadLibProvider() LuaLoadLibProvider {
	return vm.loadLibProvider
}

// Process provider

// SetProcessProvider sets the process provider for this VM.
// When set, the exec module becomes available via stdlib.Open.
func (vm *VM) SetProcessProvider(provider LuaProcessProvider) {
	vm.processProvider = provider
	vm.registerProvider(provider)
}

// ProcessProvider returns the current process provider, or nil if none is set.
func (vm *VM) ProcessProvider() LuaProcessProvider {
	return vm.processProvider
}

// Print/warn provider

// SetPrintProvider sets the print/warn output provider for this VM.
// When set, Print() and Warn() delegate to the provider instead of
// writing to stdout/stderr (or the capture buffer).
func (vm *VM) SetPrintProvider(provider LuaPrintProvider) {
	vm.printProvider = provider
	vm.registerProvider(provider)
}

// PrintProvider returns the current print provider, or nil if none is set.
func (vm *VM) PrintProvider() LuaPrintProvider {
	return vm.printProvider
}

// SetWarnEnabled sets whether warn() produces output.
// This is the per-VM equivalent of warn("@on")/"@off".
func (vm *VM) SetWarnEnabled(enabled bool) {
	vm.warnEnabled = enabled
}

// WarnEnabled returns whether warn() output is enabled for this VM.
func (vm *VM) WarnEnabled() bool {
	return vm.warnEnabled
}

// GCMode returns the current GC mode name ("incremental" or "generational").
func (vm *VM) GCMode() string {
	return vm.gcMode
}

// SetGCMode sets the GC mode name and returns the previous mode.
func (vm *VM) SetGCMode(mode string) string {
	prev := vm.gcMode
	vm.gcMode = mode
	return prev
}

// GCRunning returns whether the GC is in "running" state.
func (vm *VM) GCRunning() bool {
	return vm.gcRunning
}

// SetGCRunning sets the GC running state.
func (vm *VM) SetGCRunning(running bool) {
	vm.gcRunning = running
}

// Context and limits

// SetContext sets the context for cooperative cancellation.
func (vm *VM) SetContext(ctx context.Context) {
	vm.ctx = ctx
}

// Context returns the current context, or nil if none is set.
func (vm *VM) Context() context.Context {
	return vm.ctx
}

// SetLimits sets execution limits on the VM.
func (vm *VM) SetLimits(limits Limits) {
	vm.limits = limits
}

// GetLimits returns the current execution limits.
func (vm *VM) GetLimits() Limits {
	return vm.limits
}

// SetMaxMetaDepth sets the maximum __index/__newindex chain depth.
// Values <= 0 reset to the default (DefaultMaxMetaDepth).
func (vm *VM) SetMaxMetaDepth(n int) {
	if n <= 0 {
		vm.limits.MaxMetaDepth = 0
	} else {
		vm.limits.MaxMetaDepth = n
	}
}

// MaxMetaDepth returns the effective maximum __index/__newindex chain depth.
// Returns DefaultMaxMetaDepth if no custom value has been set.
func (vm *VM) MaxMetaDepth() int {
	if vm.limits.MaxMetaDepth <= 0 {
		return DefaultMaxMetaDepth
	}
	return vm.limits.MaxMetaDepth
}

// InstructionCount returns the current checkpoint visit count.
func (vm *VM) InstructionCount() int64 {
	return vm.instrCount
}

// ResetInstructionCount resets the checkpoint visit counter to zero.
func (vm *VM) ResetInstructionCount() {
	vm.instrCount = 0
}
