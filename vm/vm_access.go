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
}

// IoProvider returns the current IO provider, or nil if none is set.
func (vm *VM) IoProvider() LuaIoProvider {
	return vm.ioProvider
}

// SetOsProvider sets the OS provider for this VM.
func (vm *VM) SetOsProvider(provider LuaOsProvider) {
	vm.osProvider = provider
}

// OsProvider returns the current OS provider, or nil if none is set.
func (vm *VM) OsProvider() LuaOsProvider {
	return vm.osProvider
}

// SetDebugProvider sets the debug provider for this VM.
func (vm *VM) SetDebugProvider(provider LuaDebugProvider) {
	vm.debugProvider = provider
}

// DebugProvider returns the current debug provider, or nil if none is set.
func (vm *VM) DebugProvider() LuaDebugProvider {
	return vm.debugProvider
}

// SetChanProvider sets the channel provider for this VM.
func (vm *VM) SetChanProvider(provider LuaChanProvider) {
	vm.chanProvider = provider
}

// ChanProvider returns the current channel provider, or nil if none is set.
func (vm *VM) ChanProvider() LuaChanProvider {
	return vm.chanProvider
}

// SetTimeProvider sets the time provider for this VM.
func (vm *VM) SetTimeProvider(provider LuaTimeProvider) {
	vm.timeProvider = provider
}

// TimeProvider returns the current time provider, or nil if none is set.
func (vm *VM) TimeProvider() LuaTimeProvider {
	return vm.timeProvider
}

// Print/warn provider

// SetPrintProvider sets the print/warn output provider for this VM.
// When set, Print() and Warn() delegate to the provider instead of
// writing to stdout/stderr (or the capture buffer).
func (vm *VM) SetPrintProvider(provider LuaPrintProvider) {
	vm.printProvider = provider
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
