package stdlib

import (
	"fmt"
	"strings"

	"github.com/iceisfun/golua/vm"
)

// openDebug registers the diagnostic debug table if a DebugProvider is set.
func openDebug(v *vm.VM) {
	provider := v.DebugProvider()
	if provider == nil {
		return
	}

	caps := provider.Capabilities()
	debug := vm.NewEmptyTable()

	if caps.AllowTraceback {
		debug.SetString("traceback", vm.NewNativeFunc(luaDebugTraceback))
	}

	if caps.AllowStackDepth {
		debug.SetString("stackdepth", vm.NewNativeFunc(luaDebugStackDepth))
	}

	if caps.AllowWhere {
		debug.SetString("where", vm.NewNativeFunc(luaDebugWhere))
	}

	if caps.AllowGetInfo {
		debug.SetString("getinfo", vm.NewNativeFunc(luaDebugGetInfo))
	}

	if caps.AllowGetUpvalue {
		debug.SetString("getupvalue", vm.NewNativeFunc(luaDebugGetUpvalue))
	}

	if caps.AllowSetUpvalue {
		debug.SetString("setupvalue", vm.NewNativeFunc(luaDebugSetUpvalue))
	}

	if caps.AllowUpvalueID {
		debug.SetString("upvalueid", vm.NewNativeFunc(luaDebugUpvalueID))
	}

	if caps.AllowGetLocal {
		debug.SetString("getlocal", vm.NewNativeFunc(luaDebugGetLocal))
	}

	if caps.AllowSetLocal {
		debug.SetString("setlocal", vm.NewNativeFunc(luaDebugSetLocal))
	}

	if caps.AllowGetRegistry {
		debug.SetString("getregistry", vm.NewNativeFunc(luaDebugGetRegistry))
	}

	if caps.AllowGetMetatable {
		debug.SetString("getmetatable", vm.NewNativeFunc(luaDebugGetMetatable))
	}

	if caps.AllowSetMetatable {
		debug.SetString("setmetatable", vm.NewNativeFunc(luaDebugSetMetatable))
	}

	if caps.AllowSetHook {
		debug.SetString("sethook", vm.NewNativeFunc(luaDebugSetHook))
	}

	if caps.AllowGetHook {
		debug.SetString("gethook", vm.NewNativeFunc(luaDebugGetHook))
	}

	if caps.AllowUpvalueJoin {
		debug.SetString("upvaluejoin", vm.NewNativeFunc(luaDebugUpvalueJoin))
	}

	v.SetGlobal("debug", vm.NewTable(debug))
}

// debug.traceback([thread,] [message [, level]])
// If message is non-nil and non-string, it is returned unchanged (Lua 5.4).
func luaDebugTraceback(v *vm.VM) int {
	arg1 := v.Get(1)

	// Check if first arg is a thread (coroutine)
	if arg1.IsTable() {
		tbl := arg1.AsTable()
		if tbl != nil && tbl.IsThread() {
			targetVM := tbl.VMRef()
			// Shift args: (thread, msg, level) → (msg, level)
			msg := v.Get(2)
			if !msg.IsNil() && !msg.IsString() && !msg.IsNumber() {
				v.Set(0, msg) // non-string, non-nil, non-number: return as-is
				return 1
			}
			hasMsg := !msg.IsNil()
			msgStr := ""
			if msg.IsString() {
				msgStr = msg.AsString()
			} else if msg.IsNumber() {
				msgStr = valueToString(msg)
			}
			level := 0
			if !v.Get(3).IsNil() {
				if l, ok := v.Get(3).ToInt(); ok {
					level = int(l)
				}
			}
			if targetVM == nil || targetVM.StackDepth() == 0 {
				// Dead or never-resumed coroutine — empty traceback
				if hasMsg {
					v.Set(0, vm.NewString(msgStr+"\nstack traceback:"))
				} else {
					v.Set(0, vm.NewString("stack traceback:"))
				}
				return 1
			}
			tb := targetVM.Traceback("", level)
			if hasMsg {
				tb = msgStr + "\n" + tb
			}
			v.Set(0, vm.NewString(tb))
			return 1
		}
	}

	msg := arg1
	if !msg.IsNil() && !msg.IsString() && !msg.IsNumber() {
		v.Set(0, msg)
		return 1
	}

	hasMsg := !msg.IsNil()
	msgStr := ""
	if msg.IsString() {
		msgStr = msg.AsString()
	} else if msg.IsNumber() {
		msgStr = valueToString(msg)
	}

	level := 1
	if !v.Get(2).IsNil() {
		if l, ok := v.Get(2).ToInt(); ok {
			level = int(l)
		}
	}

	tb := v.Traceback("", level)
	if hasMsg {
		tb = msgStr + "\n" + tb
	}
	v.Set(0, vm.NewString(tb))
	return 1
}

// debug.stackdepth()
func luaDebugStackDepth(v *vm.VM) int {
	v.Set(0, vm.NewInt(int64(v.StackDepth())))
	return 1
}

// debug.where([level])
func luaDebugWhere(v *vm.VM) int {
	level := 1
	if !v.Get(1).IsNil() {
		if l, ok := v.Get(1).ToInt(); ok {
			level = int(l)
		}
	}

	source, line, ok := v.Where(level)
	if !ok {
		v.Set(0, vm.Nil)
		return 1
	}

	v.Set(0, vm.NewString(source))
	v.Set(1, vm.NewInt(int64(line)))
	return 2
}

// debug.getinfo([thread,] f [, what])
// f can be a function value or a stack level number.
// what is a string of option letters: f l n S t u L (default "flnStu").
func luaDebugGetInfo(v *vm.VM) int {
	arg1 := v.Get(1)
	var info *vm.FrameInfo
	var what string
	whatArg := ""
	whatGiven := false

	// Check if first arg is a thread (coroutine)
	targetVM := v
	fArg := arg1
	whatIdx := 2
	if arg1.IsTable() {
		tbl := arg1.AsTable()
		if tbl != nil && tbl.IsThread() {
			coVM := tbl.VMRef()
			if coVM != nil {
				targetVM = coVM
			} else {
				// Dead/never-resumed coroutine: no stack to inspect
				// Function form still works, level form returns nil
				fArg = v.Get(2)
				if fArg.IsCallable() {
					info = v.GetFuncInfo(fArg)
					if info == nil {
						v.Set(0, vm.Nil)
						return 1
					}
					what = "flnSrtu"
					if !v.Get(3).IsNil() {
						what = getString(v, 3, "debug.getinfo")
					}
					goto buildResult
				}
				v.Set(0, vm.Nil)
				return 1
			}
			fArg = v.Get(2)
			whatIdx = 3
		}
	}
	if !v.Get(whatIdx).IsNil() {
		whatArg = getString(v, whatIdx, "debug.getinfo")
		whatGiven = true
	}

	if fArg.IsCallable() {
		// debug.getinfo([thread,] func [, what])
		info = v.GetFuncInfo(fArg)
		if info == nil {
			v.Set(0, vm.Nil)
			return 1
		}
		what = "flnSrtu" // default
		if whatGiven {
			what = whatArg
		}
	} else {
		// debug.getinfo([thread,] level [, what])
		level := getInt(v, whatIdx-1, "debug.getinfo")
		if level < 0 {
			v.Set(0, vm.Nil)
			return 1
		}
		if targetVM == v {
			// Level 0 = getinfo itself (native frame at top of stack).
			info = targetVM.GetFrameInfo(int(level))
		} else {
			// For coroutines, level 0 = top of coroutine stack (no native frame offset)
			info = targetVM.GetFrameInfo(int(level))
		}
		if info == nil {
			v.Set(0, vm.Nil)
			return 1
		}
		what = "flnSrtu" // default
		if whatGiven {
			what = whatArg
		}
	}

buildResult:

	// Validate the what string ('>' is C API only, not valid at Lua level)
	for _, ch := range what {
		if !strings.ContainsRune("flnStuLr", ch) {
			callerArgError(v, whatIdx, "debug.getinfo", "invalid option")
		}
	}

	// Build the result table
	result := vm.NewEmptyTable()

	for _, ch := range what {
		switch ch {
		case 'S':
			result.SetString("source", vm.NewString(info.Source))
			result.SetString("short_src", vm.NewString(info.ShortSrc))
			result.SetString("linedefined", vm.NewInt(int64(info.LineDefined)))
			result.SetString("lastlinedefined", vm.NewInt(int64(info.LastLineDefined)))
			result.SetString("what", vm.NewString(info.What))
		case 'l':
			result.SetString("currentline", vm.NewInt(int64(info.CurrentLine)))
		case 'n':
			if info.Name != "" {
				result.SetString("name", vm.NewString(info.Name))
			}
			result.SetString("namewhat", vm.NewString(info.NameWhat))
		case 't':
			result.SetString("istailcall", vm.NewBool(info.IsTailCall))
		case 'u':
			result.SetString("nups", vm.NewInt(int64(info.NUps)))
			result.SetString("nparams", vm.NewInt(int64(info.NParams)))
			result.SetString("isvararg", vm.NewBool(info.IsVarArg))
		case 'f':
			result.SetString("func", info.Func)
		case 'r':
			result.SetString("ftransfer", vm.NewInt(int64(info.FTransfer)))
			result.SetString("ntransfer", vm.NewInt(int64(info.NTransfer)))
		case 'L':
			if info.ActiveLines != nil {
				lines := vm.NewEmptyTable()
				for line := range info.ActiveLines {
					lines.Set(vm.NewInt(int64(line)), vm.NewBool(true))
				}
				result.SetString("activelines", vm.NewTable(lines))
			}
			// For C functions, activelines is not set (remains nil in table)
		}
	}

	v.Set(0, vm.NewTable(result))
	return 1
}

// debug.getupvalue(f, up)
// Returns the name and value of upvalue #up of function f.
// Returns nil if the index is out of range.
// For native (Go) functions, always returns nil.
func luaDebugGetUpvalue(v *vm.VM) int {
	arg1 := v.Get(1)
	if !arg1.IsCallable() {
		got := arg1.Type()
		if v.ArgCount() < 1 {
			got = "no value"
		}
		callerArgError(v, 1, "debug.getupvalue", fmt.Sprintf("function expected, got %s", got))
	}

	arg2 := v.Get(2)
	idx, ok := arg2.ToInt()
	if !ok {
		callerArgError(v, 2, "debug.getupvalue", "number expected")
	}

	// Native functions have no inspectable upvalues
	if arg1.IsNativeFunc() {
		return 0
	}

	closure := arg1.AsClosure()
	if closure == nil {
		return 0
	}

	// Check index bounds (1-based)
	if idx < 1 || int(idx) > len(closure.Upvalues) {
		return 0
	}

	i := int(idx) - 1
	name := closure.Proto.Upvalues[i].Name
	val := closure.Upvalues[i].Get()

	v.Set(0, vm.NewString(name))
	v.Set(1, val)
	return 2
}

// debug.setupvalue(f, up, value)
// Sets the value of upvalue #up of function f.
// Returns the upvalue name, or nil if the index is out of range.
func luaDebugSetUpvalue(v *vm.VM) int {
	arg1 := v.Get(1)
	if !arg1.IsCallable() {
		got := arg1.Type()
		if v.ArgCount() < 1 {
			got = "no value"
		}
		callerArgError(v, 1, "debug.setupvalue", fmt.Sprintf("function expected, got %s", got))
	}

	arg2 := v.Get(2)
	idx, ok := arg2.ToInt()
	if !ok {
		callerArgError(v, 2, "debug.setupvalue", "number expected")
	}

	newVal := v.Get(3)

	if arg1.IsNativeFunc() {
		return 0
	}

	closure := arg1.AsClosure()
	if closure == nil {
		return 0
	}

	if idx < 1 || int(idx) > len(closure.Upvalues) {
		return 0
	}

	i := int(idx) - 1
	name := closure.Proto.Upvalues[i].Name
	closure.Upvalues[i].Set(newVal)

	v.Set(0, vm.NewString(name))
	return 1
}

// debug.upvalueid(f, n)
// Returns a unique identifier for the upvalue #n of function f.
// Two closures sharing the same upvalue return the same ID.
func luaDebugUpvalueID(v *vm.VM) int {
	arg1 := v.Get(1)
	if !arg1.IsCallable() {
		got := arg1.Type()
		if v.ArgCount() < 1 {
			got = "no value"
		}
		callerArgError(v, 1, "debug.upvalueid", fmt.Sprintf("function expected, got %s", got))
	}

	arg2 := v.Get(2)
	idx, ok := arg2.ToInt()
	if !ok {
		callerArgError(v, 2, "debug.upvalueid", "number expected")
	}

	// Native functions have 0 upvalues — return nil for any index
	if arg1.IsNativeFunc() {
		v.Set(0, vm.Nil)
		return 1
	}

	closure := arg1.AsClosure()
	if closure == nil {
		v.Set(0, vm.Nil)
		return 1
	}

	if idx < 1 || int(idx) > len(closure.Upvalues) {
		v.Set(0, vm.Nil)
		return 1
	}

	i := int(idx) - 1
	v.Set(0, vm.NewUpvalueID(closure.Upvalues[i]))
	return 1
}

// debug.getlocal([thread,] f, local)
// f can be a function value or a stack level number.
// When f is a function, returns only parameter names (no values).
// When thread is a coroutine, operates on that coroutine's stack.
// Returns the name and value of local variable #local at stack level.
// Negative local indices access varargs.
// Returns nil if out of range.
func luaDebugGetLocal(v *vm.VM) int {
	arg1 := v.Get(1)

	// Check if first arg is a thread (coroutine)
	if arg1.IsTable() {
		tbl := arg1.AsTable()
		if tbl != nil && tbl.IsThread() {
			arg2 := v.Get(2)
			if arg2.IsCallable() {
				arg3 := v.Get(3)
				local, ok := arg3.ToInt()
				if !ok {
					callerArgError(v, 3, "debug.getlocal", "number expected")
				}
				name, found := v.GetFuncLocal(arg2, int(local))
				if !found {
					v.Set(0, vm.Nil)
					return 1
				}
				v.Set(0, vm.NewString(name))
				return 1
			}
			level, ok := arg2.ToInt()
			if !ok {
				callerArgError(v, 2, "debug.getlocal", "number expected")
			}
			arg3 := v.Get(3)
			local, ok := arg3.ToInt()
			if !ok {
				callerArgError(v, 3, "debug.getlocal", "number expected")
			}
			coVM := tbl.VMRef()
			if coVM == nil || !coVM.IsValidLevel(int(level)) {
				callerArgError(v, 2, "debug.getlocal", "level out of range")
			}
			if level == 0 && coVM != v {
				v.Set(0, vm.Nil)
				return 1
			}
			// For suspended coroutines, level numbering matches the coroutine's
			// own call stack: level 0 = yield, level 1 = function that called yield.
			name, val, found := coVM.GetLocal(int(level), int(local))
			if !found {
				v.Set(0, vm.Nil)
				return 1
			}
			v.Set(0, vm.NewString(name))
			v.Set(1, val)
			return 2
		}
	}

	// Check if first arg is a function (getlocal(func, index) form)
	if arg1.IsCallable() {
		arg2 := v.Get(2)
		local, ok := arg2.ToInt()
		if !ok {
			callerArgError(v, 2, "debug.getlocal", "number expected")
		}
		name, found := v.GetFuncLocal(arg1, int(local))
		if !found {
			// Lua 5.4 returns nil (1 value) for out-of-range indices
			v.Set(0, vm.Nil)
			return 1
		}
		v.Set(0, vm.NewString(name))
		return 1
	}

	level, ok := arg1.ToInt()
	if !ok {
		got := arg1.Type()
		if v.ArgCount() < 1 {
			got = "no value"
		}
		callerArgError(v, 1, "debug.getlocal", fmt.Sprintf("number expected, got %s", got))
	}

	arg2 := v.Get(2)
	local, ok := arg2.ToInt()
	if !ok {
		callerArgError(v, 2, "debug.getlocal", "number expected")
	}

	// Validate level is in range (level 0 = getlocal itself = native frame)
	if !v.IsValidLevel(int(level)) {
		callerArgError(v, 1, "debug.getlocal", "level out of range")
	}

	name, val, found := v.GetLocal(int(level), int(local))
	if !found {
		// Lua 5.4 returns nil (1 value) for out-of-range indices
		v.Set(0, vm.Nil)
		return 1
	}

	v.Set(0, vm.NewString(name))
	v.Set(1, val)
	return 2
}

// debug.setlocal([thread,] level, local, value)
// Sets the value of local variable #local at the given stack level.
// When thread is a coroutine, operates on that coroutine's stack.
// Returns the name of the variable, or nil if out of range.
func luaDebugSetLocal(v *vm.VM) int {
	arg1 := v.Get(1)

	// Check if first arg is a thread (coroutine)
	if arg1.IsTable() {
		tbl := arg1.AsTable()
		if tbl != nil && tbl.IsThread() {
			arg2 := v.Get(2)
			level, ok := arg2.ToInt()
			if !ok {
				callerArgError(v, 2, "debug.setlocal", "number expected")
			}
			arg3 := v.Get(3)
			local, ok := arg3.ToInt()
			if !ok {
				callerArgError(v, 3, "debug.setlocal", "number expected")
			}
			newVal := v.Get(4)
			coVM := tbl.VMRef()
			if coVM == nil || !coVM.IsValidLevel(int(level)) {
				callerArgError(v, 2, "debug.setlocal", "level out of range")
			}
			// For suspended coroutines, level numbering matches the coroutine's
			// own call stack: level 0 = yield, level 1 = function that called yield.
			name, found := coVM.SetLocal(int(level), int(local), newVal)
			if !found {
				v.Set(0, vm.Nil)
				return 1
			}
			v.Set(0, vm.NewString(name))
			return 1
		}
	}

	level, ok := arg1.ToInt()
	if !ok {
		got := arg1.Type()
		if v.ArgCount() < 1 {
			got = "no value"
		}
		callerArgError(v, 1, "debug.setlocal", fmt.Sprintf("number expected, got %s", got))
	}

	arg2 := v.Get(2)
	local, ok := arg2.ToInt()
	if !ok {
		callerArgError(v, 2, "debug.setlocal", "number expected")
	}

	newVal := v.Get(3)

	if !v.IsValidLevel(int(level)) {
		callerArgError(v, 1, "debug.setlocal", "level out of range")
	}

	name, found := v.SetLocal(int(level), int(local), newVal)
	if !found {
		v.Set(0, vm.Nil)
		return 1
	}

	v.Set(0, vm.NewString(name))
	return 1
}

// debug.getregistry()
// Returns the registry table.
func luaDebugGetRegistry(v *vm.VM) int {
	v.Set(0, vm.NewTable(v.GetRegistry()))
	return 1
}

// debug.getmetatable(value)
// Returns the raw metatable of value, bypassing __metatable protection.
// Works for any type (including non-tables via type metatables).
func luaDebugGetMetatable(v *vm.VM) int {
	val := v.Get(1)
	// For threads, use type-level metatable (not per-instance)
	if val.IsTable() && val.AsTable().IsThread() {
		if mt := v.GetTypeMeta(val); mt != nil {
			v.Set(0, vm.NewTable(mt))
		} else {
			v.Set(0, vm.Nil)
		}
		return 1
	}
	if val.IsTable() {
		mt := val.AsTable().Metatable()
		if mt != nil {
			v.Set(0, vm.NewTable(mt))
		} else {
			v.Set(0, vm.Nil)
		}
		return 1
	}
	// Check userdata per-instance metatable
	if ud := val.AsUserdata(); ud != nil {
		if mt := ud.Metatable(); mt != nil {
			v.Set(0, vm.NewTable(mt))
			return 1
		}
		v.Set(0, vm.Nil)
		return 1
	}
	// Non-table: return type metatable
	if mt := v.GetTypeMeta(val); mt != nil {
		v.Set(0, vm.NewTable(mt))
		return 1
	}
	v.Set(0, vm.Nil)
	return 1
}

// debug.setmetatable(value, table)
// Sets the metatable of value, bypassing __metatable protection.
// For non-table values, sets the type metatable (affects all values of that type).
// Returns the first argument (value).
func luaDebugSetMetatable(v *vm.VM) int {
	val := v.Get(1)
	mt := v.Get(2)

	var mtTable vm.LuaTable
	if mt.IsTable() {
		mtTable = mt.AsTable()
	} else if !mt.IsNil() {
		callerArgError(v, 2, "debug.setmetatable", "nil or table expected")
	}

	if val.IsTable() && !val.AsTable().IsThread() {
		val.AsTable().SetMetatable(mtTable)
	} else if ud := val.AsUserdata(); ud != nil {
		ud.SetMetatable(mtTable)
	} else {
		v.SetTypeMeta(val, mtTable)
	}

	v.Set(0, val)
	return 1
}

// debug.sethook([thread,] hook, mask [, count])
// Sets a hook function. mask is a string with 'c' (call), 'r' (return), 'l' (line).
// count is the instruction count for count hooks.
// Call with no arguments to remove the hook.
func luaDebugSetHook(v *vm.VM) int {
	arg1 := v.Get(1)

	// Check if first arg is a thread (coroutine)
	targetVM := v
	hookIdx := 1
	if arg1.IsTable() {
		tbl := arg1.AsTable()
		if tbl != nil && tbl.IsThread() {
			coVM := tbl.VMRef()
			if coVM != nil {
				targetVM = coVM
			}
			hookIdx = 2
			arg1 = v.Get(hookIdx)
		}
	}

	// No arguments or nil hook arg: remove hook
	if arg1.IsNil() || v.ArgCount() < hookIdx {
		targetVM.SetHook(vm.Nil, 0, 0)
		return 0
	}

	if !arg1.IsCallable() {
		callerArgError(v, hookIdx, "debug.sethook", fmt.Sprintf("function expected, got %s", arg1.Type()))
	}

	maskArg := v.Get(hookIdx + 1)
	if !maskArg.IsString() {
		callerArgError(v, hookIdx+1, "debug.sethook", "string expected")
	}
	maskStr := maskArg.AsString()

	var mask byte
	for _, ch := range maskStr {
		switch ch {
		case 'c':
			mask |= vm.HookMaskCall
		case 'r':
			mask |= vm.HookMaskReturn
		case 'l':
			mask |= vm.HookMaskLine
		default:
			// Unknown characters are ignored (Lua 5.4 behavior)
		}
	}

	count := 0
	countArg := v.Get(hookIdx + 2)
	if !countArg.IsNil() {
		if c, ok := countArg.ToInt(); ok {
			count = int(c)
			if count > 0 {
				mask |= vm.HookMaskCount
			}
		}
	}

	targetVM.SetHook(arg1, mask, count)
	return 0
}

// debug.upvaluejoin(f1, n1, f2, n2)
// Makes the n1-th upvalue of f1 refer to the same storage as the n2-th upvalue of f2.
func luaDebugUpvalueJoin(v *vm.VM) int {
	f1 := v.Get(1)
	if !f1.IsCallable() {
		got := f1.Type()
		if v.ArgCount() < 1 {
			got = "no value"
		}
		callerArgError(v, 1, "debug.upvaluejoin", fmt.Sprintf("function expected, got %s", got))
	}
	n1, ok := v.Get(2).ToInt()
	if !ok {
		callerArgError(v, 2, "debug.upvaluejoin", "number expected")
	}
	f2 := v.Get(3)
	if !f2.IsCallable() {
		got := f2.Type()
		if v.ArgCount() < 3 {
			got = "no value"
		}
		callerArgError(v, 3, "debug.upvaluejoin", fmt.Sprintf("function expected, got %s", got))
	}
	n2, ok := v.Get(4).ToInt()
	if !ok {
		callerArgError(v, 4, "debug.upvaluejoin", "number expected")
	}

	// Native functions have 0 upvalues — any index is invalid
	c1 := f1.AsClosure()
	if c1 == nil || n1 < 1 || int(n1) > len(c1.Upvalues) {
		callerArgError(v, 2, "debug.upvaluejoin", "invalid upvalue index")
	}
	c2 := f2.AsClosure()
	if c2 == nil || n2 < 1 || int(n2) > len(c2.Upvalues) {
		callerArgError(v, 4, "debug.upvaluejoin", "invalid upvalue index")
	}

	c1.Upvalues[int(n1)-1] = c2.Upvalues[int(n2)-1]
	return 0
}

// debug.gethook([thread])
// Returns the current hook function, mask string, and count.
func luaDebugGetHook(v *vm.VM) int {
	// Check if first arg is a thread (coroutine)
	targetVM := v
	arg1 := v.Get(1)
	if arg1.IsTable() {
		tbl := arg1.AsTable()
		if tbl != nil && tbl.IsThread() {
			coVM := tbl.VMRef()
			if coVM != nil {
				targetVM = coVM
			} else {
				// Dead/never-resumed coroutine: no hook
				v.Set(0, vm.Nil)
				v.Set(1, vm.Nil)
				v.Set(2, vm.Nil)
				return 3
			}
		}
	}

	hookFunc, mask, count := targetVM.GetHook()

	v.Set(0, hookFunc)

	// Lua 5.4: when no hook is set, gethook returns nil, nil, nil (not nil, "", 0).
	if hookFunc == vm.Nil {
		v.Set(1, vm.Nil)
		v.Set(2, vm.Nil)
		return 3
	}

	// Build mask string
	var maskStr string
	if mask&vm.HookMaskCall != 0 {
		maskStr += "c"
	}
	if mask&vm.HookMaskReturn != 0 {
		maskStr += "r"
	}
	if mask&vm.HookMaskLine != 0 {
		maskStr += "l"
	}

	v.Set(1, vm.NewString(maskStr))
	v.Set(2, vm.NewInt(int64(count)))
	return 3
}
