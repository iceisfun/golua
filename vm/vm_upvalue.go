package vm

import "fmt"

// Upvalue management

func (vm *VM) findOrCreateUpvalue(stackIdx int) *Upvalue {
	// Look for existing open upvalue at this index
	for _, uv := range vm.openUpvalues {
		if uv.stackIdx == stackIdx {
			return uv
		}
	}

	// Create new open upvalue
	uv := NewOpenUpvalue(vm, stackIdx)
	vm.openUpvalues = append(vm.openUpvalues, uv)
	return uv
}

func (vm *VM) closeUpvalues(level int) {
	// Close all upvalues with stack index >= level
	remaining := vm.openUpvalues[:0]
	for _, uv := range vm.openUpvalues {
		if uv.stackIdx >= level {
			uv.Close()
		} else {
			remaining = append(remaining, uv)
		}
	}
	vm.openUpvalues = remaining

	// Call __close metamethod on TBC variables in reverse order
	// (most recently declared first).
	// Remove entries from tbcVars BEFORE calling __close to prevent
	// double-close if the handler panics (the panic would be caught by
	// ProtectedCall's recover, which also processes tbcVars).
	var tbcToClose []int
	var remainingTBC []int
	for _, idx := range vm.tbcVars {
		if idx >= level {
			tbcToClose = append(tbcToClose, idx)
		} else {
			remainingTBC = append(remainingTBC, idx)
		}
	}
	vm.tbcVars = remainingTBC
	// Call in reverse order (most recently declared first).
	// Each call is protected so that errors in __close don't prevent
	// other handlers from running. The last error is re-raised.
	vm.callCloseHandlers(tbcToClose, Nil)
}

// callCloseHandlers calls __close metamethods on a list of TBC variable indices
// in reverse order. Each call is individually protected so that all handlers run
// even if one errors. If any handler errors, the last error is re-raised after
// all handlers have been called.
func (vm *VM) callCloseHandlers(indices []int, errVal Value) {
	var lastPanic interface{}
	for i := len(indices) - 1; i >= 0; i-- {
		func() {
			defer func() {
				if r := recover(); r != nil {
					lastPanic = r
					// Update errVal for subsequent handlers (Lua 5.4 behavior:
					// __close receives the most recent error)
					if le, ok := r.(*LuaError); ok {
						errVal = le.Value
					} else {
						errVal = NewString(fmt.Sprintf("%v", r))
					}
					// Restore call stack in case the panic left it dirty
					// (the handler's frames should not persist)
				}
			}()
			vm.callCloseMetamethod(indices[i], errVal)
		}()
	}
	if lastPanic != nil {
		panic(lastPanic)
	}
}

// CloseAllTBC closes all to-be-closed variables in the VM.
// Used when a coroutine is closed externally (coroutine.close).
func (vm *VM) CloseAllTBC() {
	if len(vm.tbcVars) == 0 {
		return
	}
	tbcToClose := make([]int, len(vm.tbcVars))
	copy(tbcToClose, vm.tbcVars)
	vm.tbcVars = nil
	// Protect each handler individually; ignore panics since the
	// coroutine is being terminated anyway.
	for i := len(tbcToClose) - 1; i >= 0; i-- {
		func() {
			defer func() { recover() }()
			vm.callCloseMetamethod(tbcToClose[i], Nil)
		}()
	}
}

// callCloseMetamethod calls the __close metamethod on a TBC variable.
// errVal is the error object (Nil for normal exit, the Lua error value on error paths).
func (vm *VM) callCloseMetamethod(stackIdx int, errVal Value) {
	val := vm.stack[stackIdx]
	if !val.IsTable() {
		return
	}
	t := val.AsTable()
	mt := t.Metatable()
	if mt == nil {
		return
	}
	closeFunc := mt.Get(metaClose)
	if closeFunc.IsNil() {
		return
	}
	vm.callMetamethod(closeFunc, val, errVal)
}
