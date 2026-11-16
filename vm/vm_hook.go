package vm

import (
	"github.com/iceisfun/golua/v2/compiler"
)

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
//
// As in reference Lua, an error raised by the hook propagates like any other
// error and can be caught by a Lua pcall. A host that needs an uncatchable
// instruction or time limit must use WithLimits or WithContext instead of
// raising from a hook.
func (vm *VM) SetHook(fn Value, mask byte, count int) {
	vm.hookFunc = fn
	vm.hookMask = mask
	vm.hookCount = count
	vm.hookCounter = count
	// Match Lua 5.4: when sethook is called, L->oldpc already tracks the
	// current instruction's pc, so the line hook doesn't re-fire for the
	// same line. We approximate this by setting lastHookLine to the current
	// line of the calling Lua frame (if any), so the calling function's
	// current line is not spuriously reported as "new".
	if mask&HookMaskLine != 0 && len(vm.callStack) > 0 {
		// Walk up the call stack to find the nearest Lua frame
		for i := len(vm.callStack) - 1; i >= 0; i-- {
			frame := &vm.callStack[i]
			if frame.closure != nil && frame.pc > 0 {
				proto := frame.closure.Proto
				pc := frame.pc - 1
				if pc >= 0 && pc < len(proto.Lines) {
					vm.lastHookLine = proto.Lines[pc]
					vm.lastHookPC = pc
					return
				}
			}
		}
	}
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

	// Build args. Like ldblib's hookf, always pass two: the line number for
	// a line event, nil for every other event.
	lineArg := Nil
	if event == hookEventLine && line >= 0 {
		lineArg = NewInt(int64(line))
	}
	args := []Value{NewString(event), lineArg}

	// Call in non-yieldable context
	exit := vm.EnterNonYieldable()
	defer exit()

	var savedFTransfer, savedNTransfer int
	hasTopFrame := len(vm.callStack) > 0
	if hasTopFrame && (event == hookEventLine || event == hookEventCount) {
		top := &vm.callStack[len(vm.callStack)-1]
		savedFTransfer, savedNTransfer = top.ftransfer, top.ntransfer
		top.ftransfer, top.ntransfer = 0, 0
		defer func() {
			top.ftransfer, top.ntransfer = savedFTransfer, savedNTransfer
		}()
	}

	savedCallName := vm.pendingCallName
	savedCallNameWhat := vm.pendingCallNameWhat
	vm.pendingCallName = "?"
	vm.pendingCallNameWhat = "hook"
	defer func() {
		vm.pendingCallName = savedCallName
		vm.pendingCallNameWhat = savedCallNameWhat
	}()

	// Ensure vm.top covers the current frame's full register space so the
	// hook's call frame doesn't overlap with active registers. After a CALL
	// with c=0 (variable results), vm.top may be lowered below
	// frame.base + MaxStack; the hook function's vm.call cleanup would then
	// clear slots within the caller's register range, clobbering live values.
	savedTop := vm.top
	if hasTopFrame {
		top := &vm.callStack[len(vm.callStack)-1]
		if top.closure != nil {
			minTop := top.base + top.closure.Proto.MaxStack
			if vm.top < minTop {
				vm.top = minTop
			}
		}
	}
	defer func() { vm.top = savedTop }()

	// Call the hook function directly (unprotected) so errors propagate.
	// As in reference Lua, where luaD_hook runs the hook as an ordinary
	// call, an error raised by the hook unwinds to the nearest protected
	// call: a pcall/xpcall around the hooked code catches it.
	vm.callUnprotected(vm.hookFunc, args)
}

// fireCallHook fires a "call" hook event.
func (vm *VM) fireCallHook() {
	if vm.hookMask&HookMaskCall != 0 && !vm.inHook {
		vm.fireHook(hookEventCall, 0)
	}
}

// fireReturnHook fires a "return" hook event.
func (vm *VM) fireReturnHook() {
	if vm.hookMask&HookMaskReturn != 0 && !vm.inHook {
		vm.fireHook(hookEventReturn, 0)
	}
}

// nativeHookArgs returns the arguments of the native frame at base as they
// stand after its call hook, which may have changed them with
// debug.setlocal. With no hook set it returns args unchanged.
func (vm *VM) nativeHookArgs(base int, args []Value) []Value {
	if vm.hookMask == 0 {
		return args
	}
	return append([]Value(nil), vm.stack[base+1:base+1+len(args)]...)
}

// fireNativeReturnHook fires the return hook for the native frame on top of
// the call stack, which was called with args and returned results. The
// native wrote its results over its own function and argument slots, so
// first lay the stack out as reference Lua leaves it, [args..., results...],
// and point ftransfer/ntransfer at the results so the hook can reach them
// with debug.getlocal (and change them with debug.setlocal).
func (vm *VM) fireNativeReturnHook(args, results []Value) {
	if vm.hookMask&HookMaskReturn == 0 || vm.inHook {
		return
	}
	nf := &vm.callStack[len(vm.callStack)-1]
	retStart := nf.base + 1 + len(args)
	vm.ensureStack(retStart + len(results) + stackSafetyMargin)
	copy(vm.stack[nf.base+1:], args)
	copy(vm.stack[retStart:], results)
	vm.top = retStart + len(results)
	if len(results) > 0 {
		nf.ftransfer = retStart - nf.base
		nf.ntransfer = len(results)
	} else {
		nf.ftransfer = 0
		nf.ntransfer = 0
	}
	vm.fireReturnHook()
	copy(results, vm.stack[retStart:])
}

// fireTailCallHook fires a "tail call" hook event.
func (vm *VM) fireTailCallHook() {
	if vm.hookMask&HookMaskCall != 0 && !vm.inHook {
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
			vm.fireHook(hookEventCount, -1)
			fired = true
		}
	}

	// Line hook — fire when the line changes OR on backward jumps (loops).
	// Lua 5.4 fires when: newpc == 0 || newpc <= oldpc || changedline(p, oldpc, newpc)
	// We approximate this with pc-based backward jump detection.
	if vm.hookMask&HookMaskLine != 0 {
		if pc >= 0 && pc < len(proto.Lines) {
			line := proto.Lines[pc]
			backwardJump := pc <= vm.lastHookPC
			vm.lastHookPC = pc
			if line > 0 && (line != vm.lastHookLine || backwardJump) {
				vm.lastHookLine = line
				vm.fireHook(hookEventLine, line)
				fired = true
			}
		} else if len(proto.Lines) == 0 && pc == 0 {
			// Stripped function: fire line hook with -1 (nil in Lua)
			// on the first instruction only. Lua 5.4 fires once per
			// call because the jump-back check succeeds on entry but
			// changedline returns 0 for subsequent instructions.
			vm.lastHookLine = -1
			vm.fireHook(hookEventLine, -1)
			fired = true
		}
	}

	return fired
}
