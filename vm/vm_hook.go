package vm

import (
	"fmt"

	"github.com/iceisfun/golua/compiler"
)

// luaHookError wraps a panic from a debug hook function. In Lua 5.4, hook
// errors are uncatchable by pcall/xpcall — they propagate through all
// protected frames (similar to os.exit). ProtectedCall detects this sentinel
// and re-panics instead of recovering.
type luaHookError struct {
	original interface{} // the underlying panic value from the hook
}

func (e *luaHookError) Error() string {
	return fmt.Sprintf("%v", e.original)
}

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

	// Build args
	args := []Value{NewString(event)}
	if event == hookEventLine || event == hookEventCount {
		if event == hookEventCount {
			args = append(args, Nil)
		} else if line >= 0 {
			args = append(args, NewInt(int64(line)))
		} else {
			args = append(args, Nil)
		}
	}

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
	// In Lua 5.4, hook errors are uncatchable by pcall/xpcall — they
	// propagate through all protected frames. We wrap any panic from the
	// hook in a luaHookError sentinel so ProtectedCall re-panics it.
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Wrap the original panic in a hook error sentinel
				panic(&luaHookError{original: r})
			}
		}()
		vm.callUnprotected(vm.hookFunc, args)
	}()
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
