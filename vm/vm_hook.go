package vm

import "github.com/iceisfun/golua/compiler"

// Hook mask constants (bitmask for fast checking)
const (
	HookMaskCall   byte = 1
	HookMaskReturn byte = 2
	HookMaskLine   byte = 4
	HookMaskCount  byte = 8
)

// Hook event strings
const (
	hookEventCall     = "call"
	hookEventReturn   = "return"
	hookEventTailCall = "tail call"
	hookEventLine     = "line"
	hookEventCount    = "count"
)

// SetHook configures the debug hook.
func (vm *VM) SetHook(fn Value, mask byte, count int) {
	vm.hookFunc = fn
	vm.hookMask = mask
	vm.hookCount = count
	vm.hookCounter = count
	vm.lastHookLine = -1
}

// GetHook returns the current hook function, mask, and count.
func (vm *VM) GetHook() (Value, byte, int) {
	return vm.hookFunc, vm.hookMask, vm.hookCount
}

// fireHook calls the hook function with the given event and optional line number.
// It is guarded against re-entrancy via inHook.
func (vm *VM) fireHook(event string, line int) {
	if vm.inHook || vm.hookFunc.IsNil() {
		return
	}
	vm.inHook = true
	defer func() { vm.inHook = false }()

	// Build args
	args := []Value{NewString(event)}
	if event == hookEventLine || event == hookEventCount {
		args = append(args, NewInt(int64(line)))
	}

	// Call in non-yieldable context
	exit := vm.EnterNonYieldable()
	defer exit()

	// Call the hook function directly (unprotected) so errors propagate
	// to the enclosing pcall/xpcall, matching Lua 5.4 semantics.
	vm.callUnprotected(vm.hookFunc, args)
}

// fireCallHook fires a "call" hook event.
func (vm *VM) fireCallHook() {
	if vm.hookMask&HookMaskCall != 0 && !vm.inHook {
		vm.lastHookLine = -1
		vm.fireHook(hookEventCall, 0)
	}
}

// fireReturnHook fires a "return" hook event.
func (vm *VM) fireReturnHook() {
	if vm.hookMask&HookMaskReturn != 0 && !vm.inHook {
		vm.lastHookLine = -1
		vm.fireHook(hookEventReturn, 0)
	}
}

// fireTailCallHook fires a "tail call" hook event.
func (vm *VM) fireTailCallHook() {
	if vm.hookMask&HookMaskCall != 0 && !vm.inHook {
		vm.lastHookLine = -1
		vm.fireHook(hookEventTailCall, 0)
	}
}

// checkLineCountHooks checks and fires line and count hooks.
// Called from the main instruction loop. Returns true if a hook was fired
// (caller should re-fetch frame/proto/code).
func (vm *VM) checkLineCountHooks(proto *compiler.Proto, pc int) bool {
	if vm.inHook {
		return false
	}

	fired := false

	// Count hook
	if vm.hookMask&HookMaskCount != 0 {
		vm.hookCounter--
		if vm.hookCounter <= 0 {
			vm.hookCounter = vm.hookCount
			line := 0
			if pc >= 0 && pc < len(proto.Lines) {
				line = proto.Lines[pc]
			}
			vm.fireHook(hookEventCount, line)
			fired = true
		}
	}

	// Line hook
	if vm.hookMask&HookMaskLine != 0 {
		if pc >= 0 && pc < len(proto.Lines) {
			line := proto.Lines[pc]
			if line != vm.lastHookLine && line > 0 {
				vm.lastHookLine = line
				vm.fireHook(hookEventLine, line)
				fired = true
			}
		}
	}

	return fired
}
