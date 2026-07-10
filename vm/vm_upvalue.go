package vm

import "fmt"

// Upvalue management

// findOpenUpvalue returns the open upvalue already bound to the given stack
// index, or nil if no closure has captured that slot yet.
func (vm *VM) findOpenUpvalue(stackIdx int) *Upvalue {
	for _, uv := range vm.openUpvalues {
		if uv.stackIdx == stackIdx {
			return uv
		}
	}
	return nil
}

// findOrCreateUpvalue returns an existing open upvalue for the given stack
// index, or creates a new one if none exists yet.
func (vm *VM) findOrCreateUpvalue(stackIdx int) *Upvalue {
	if uv := vm.findOpenUpvalue(stackIdx); uv != nil {
		return uv
	}

	// Create new open upvalue
	uv := NewOpenUpvalue(vm, stackIdx)
	vm.openUpvalues = append(vm.openUpvalues, uv)
	return uv
}

// closeUpvalues closes all open upvalues at or above the given stack level,
// then calls __close metamethods on any to-be-closed variables in that range.
func (vm *VM) closeUpvalues(level int) {
	vm.closeUpvaluesWithError(level, Nil, false)
}

func (vm *VM) closeUpvaluesWithError(level int, errVal Value, hasError bool) {
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

	// Call __close metamethods on TBC variables in reverse order
	// (most recently declared first). Unlike callCloseHandlers (used in
	// protected/error-recovery contexts), handlers here are NOT individually
	// protected: if one errors, the panic propagates immediately and the
	// chain stops. This matches Lua 5.4 behavior where __close errors in
	// unprotected scope exit stop the chain. If a ProtectedCall is active,
	// its recovery will pick up any remaining TBC vars still in vm.tbcVars
	// and close them with callCloseHandlers (which IS individually protected).
	//
	// We remove each entry from vm.tbcVars one at a time before calling
	// its handler, so that (a) already-called entries aren't re-closed by
	// ProtectedCall recovery, and (b) not-yet-called entries remain in
	// vm.tbcVars for ProtectedCall recovery to find.

	// Partition: find which entries are >= level and which are below.
	// We'll process the >= level ones in reverse but need to track them
	// by their position in vm.tbcVars.
	var tbcIndices []int // positions in vm.tbcVars that are >= level
	for i, idx := range vm.tbcVars {
		if idx >= level {
			tbcIndices = append(tbcIndices, i)
		}
	}
	// Process in reverse order (most recently declared first).
	for j := len(tbcIndices) - 1; j >= 0; j-- {
		pos := tbcIndices[j]
		stackIdx := vm.tbcVars[pos]
		// Remove this entry from vm.tbcVars before calling the handler.
		vm.tbcVars = append(vm.tbcVars[:pos], vm.tbcVars[pos+1:]...)
		// Update remaining positions in tbcIndices since we shifted elements.
		for k := 0; k < j; k++ {
			if tbcIndices[k] > pos {
				tbcIndices[k]--
			}
		}
		vm.callCloseMetamethod(stackIdx, errVal, hasError)
	}
}

// callCloseHandlers calls __close metamethods on a list of TBC variable indices
// in reverse order. Each call is individually protected so that all handlers run
// even if one errors. If any handler errors, the last error is re-raised after
// all handlers have been called.
func (vm *VM) callCloseHandlers(indices []int, errVal Value, hasError bool, useMsgHandler bool) {
	// Guard against unbounded Go-stack recursion. When an erroring __close
	// handler declares its own to-be-closed variable that errors again, the Lua
	// callStack is unwound by the per-handler recover below, so checkCallDepth
	// never fires — but each recovery recurses back into callCloseHandlers on
	// the Go stack, overflowing the host process (a host crash).
	// Bound the close-chain depth (the counter is shared with coroutine.close)
	// and raise a catchable "C stack overflow" instead, matching reference Lua.
	if vm.EnterCloseChain() {
		vm.ExitCloseChain()
		panic(&LuaError{Value: NewString(vm.runtimeError("%s", "C stack overflow").Error())})
	}
	defer vm.ExitCloseChain()

	var lastPanic interface{}
	for i := len(indices) - 1; i >= 0; i-- {
		savedTbcLen := len(vm.tbcVars)
		savedCallStackLen := len(vm.callStack)
		func() {
			savedTop := vm.top
			defer func() {
				vm.top = savedTop
				if r := recover(); r != nil {
					lastPanic = r
					// A handler errored: subsequent handlers now run in an error
					// context (they receive an error object even if it is nil).
					hasError = true
					// Update errVal for subsequent handlers (Lua 5.4 behavior:
					// __close receives the most recent error)
					prevErrVal := errVal
					rawErrVal := errVal
					if le, ok := r.(*LuaError); ok {
						rawErrVal = le.Value
					} else {
						rawErrVal = NewString(fmt.Sprintf("%v", r))
					}
					errVal = rawErrVal

					// Save the call stack snapshot for the message handler.
					// The last error's stack will be used by ProtectedCall
					// to call xpcall's message handler.
					if useMsgHandler && !vm.msgHandler.IsNil() && !rawErrVal.RawEqual(prevErrVal) {
						vm.lastErrorCallStack = make([]callFrame, len(vm.callStack))
						copy(vm.lastErrorCallStack, vm.callStack)
						vm.callMsgHandler(vm.msgHandler, rawErrVal, vm.lastErrorCallStack)
						if vm.msgHandlerUsed {
							errVal = vm.msgHandlerResult
						}
					}

					// Restore call stack in case the panic left it dirty
					if len(vm.callStack) > savedCallStackLen {
						vm.callStack = vm.callStack[:savedCallStackLen]
					}
					// Close TBC variables created inside the failed handler.
					// A __close function may itself declare TBC variables; if it
					// errors, those inner TBC vars must be closed with the error
					// propagated as their msg argument.
					if len(vm.tbcVars) > savedTbcLen {
						innerTBC := make([]int, len(vm.tbcVars)-savedTbcLen)
						copy(innerTBC, vm.tbcVars[savedTbcLen:])
						vm.tbcVars = vm.tbcVars[:savedTbcLen]
						func() {
							defer func() {
								if innerR := recover(); innerR != nil {
									lastPanic = innerR
									if le, ok := innerR.(*LuaError); ok {
										errVal = le.Value
									} else {
										errVal = NewString(fmt.Sprintf("%v", innerR))
									}
								}
							}()
							vm.callCloseHandlers(innerTBC, errVal, hasError, useMsgHandler)
						}()
					}
				}
			}()
			vm.callCloseMetamethod(indices[i], errVal, hasError)
		}()
	}
	// Re-raise the last __close handler error after all handlers have run.
	// This panic propagates to ProtectedCall's recover(), which converts it
	// to an error return. The re-raise ensures the error is not silently lost.
	if lastPanic != nil {
		panic(lastPanic)
	}
}

// CloseAllTBC closes all to-be-closed variables in the VM.
// Used when a coroutine is closed externally (coroutine.close).
// If any __close handler panics, the last panic is re-raised after
// all handlers have run, so the error propagates to runCoroutine.
func (vm *VM) CloseAllTBC() {
	if len(vm.tbcVars) == 0 {
		return
	}
	// coroutine.close runs __close handlers against a dead/suspended coroutine.
	// Lua 5.4 does not expose the coroutine's previously suspended frames to
	// debug.getinfo/debug.traceback from inside those handlers, so temporarily
	// detach the old call stack while the close handlers execute.
	// Use an empty slice (not nil) so append in callMetamethod works correctly,
	// and preserve vm.top so new frames start above the TBC variables.
	savedCallStack := vm.callStack
	vm.callStack = vm.callStack[:0]
	defer func() {
		vm.callStack = savedCallStack
	}()
	// Run __close handlers in a non-yieldable context (matches Lua 5.4).
	// coroutine.close runs handlers where yield is not allowed.
	exit := vm.EnterNonYieldable()
	defer exit()
	tbcToClose := make([]int, len(vm.tbcVars))
	copy(tbcToClose, vm.tbcVars)
	vm.tbcVars = nil
	// Protect each handler individually so all handlers run even if
	// one errors. The last error is re-raised so coroutine.close can
	// report it. Chain errVal through handlers (Lua 5.4 behavior).
	var lastPanic interface{}
	errVal := Nil
	hasError := false
	for i := len(tbcToClose) - 1; i >= 0; i-- {
		func() {
			defer func() {
				if r := recover(); r != nil {
					lastPanic = r
					hasError = true
					if le, ok := r.(*LuaError); ok {
						errVal = le.Value
					} else {
						errVal = NewString(fmt.Sprintf("%v", r))
					}
					// Close any to-be-closed variables the failed handler
					// declared in its own scope. This mirrors normal scope-exit
					// unwinding: a recursive erroring __close chain then hits the
					// close-depth guard in callCloseHandlers and surfaces
					// "C stack overflow" like reference Lua, instead of dropping
					// the inner variables and returning the innermost error.
					if len(vm.tbcVars) > 0 {
						inner := make([]int, len(vm.tbcVars))
						copy(inner, vm.tbcVars)
						vm.tbcVars = nil
						func() {
							defer func() {
								if ir := recover(); ir != nil {
									lastPanic = ir
									if le, ok := ir.(*LuaError); ok {
										errVal = le.Value
									} else {
										errVal = NewString(fmt.Sprintf("%v", ir))
									}
								}
							}()
							vm.callCloseHandlers(inner, errVal, hasError, false)
						}()
					}
				}
			}()
			vm.callCloseMetamethod(tbcToClose[i], errVal, hasError)
		}()
	}
	if lastPanic != nil {
		panic(lastPanic)
	}
}

// callCloseMetamethod calls the __close metamethod on a TBC variable.
// errVal is the error object on error paths. hasError distinguishes a normal
// close (no error object; __close receives only the object) from an error close
// (__close receives object plus errVal, even when errVal is itself nil — e.g.
// error(nil)). This mirrors Lua 5.5 lfunc.c callclosemethod, which pushes the
// error argument only when err != NULL.
func (vm *VM) callCloseMetamethod(stackIdx int, errVal Value, hasError bool) {
	val := vm.stack[stackIdx]
	// nil and false are always OK (no __close needed)
	if val.IsNil() || (val.IsBool() && !val.AsBool()) {
		return
	}
	// Look for __close in table or userdata metatable
	var closeFunc Value
	if val.IsTable() {
		t := val.AsTable()
		mt := t.Metatable()
		if mt == nil {
			// Check type-level metatable (threads have no per-instance metatable)
			mt = vm.GetTypeMeta(val)
		}
		if mt != nil {
			closeFunc = mt.Get(metaClose)
		}
	} else if ud := val.AsUserdata(); ud != nil {
		mt := ud.Metatable()
		if mt != nil {
			closeFunc = mt.Get(metaClose)
		}
	} else {
		// Check type-level metatable (e.g. debug.setmetatable(0, {__close=...}))
		typeMT := vm.GetTypeMeta(val)
		if typeMT != nil {
			closeFunc = typeMT.Get(metaClose)
		}
	}
	if !closeFunc.IsNil() {
		vm.pendingSuppressTracebackName = hasError || len(vm.callStack) == 0
		// Lua 5.5: __close receives only the object on a normal close, and
		// (object, errobj) on an error close. On the normal path there is no
		// error object, so omit the second argument entirely (matching lfunc.c
		// callclosemethod which only pushes the error when err != NULL).
		var err error
		if hasError {
			_, err = vm.callMetamethodArgs("close", closeFunc, val, errVal)
		} else {
			_, err = vm.callMetamethodArgs("close", closeFunc, val)
		}
		if err != nil {
			panic(err.Error())
		}
		return
	}
	// Value was registered as TBC but __close is now missing (removed at
	// runtime or metatable changed). Generate a LuaError with source location
	// to match Lua 5.4 behavior.
	panic(&LuaError{Value: NewString(vm.runtimeError("attempt to call a nil value (metamethod 'close')").Error())})
}
