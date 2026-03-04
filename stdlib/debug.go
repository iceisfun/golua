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

	v.SetGlobal("debug", vm.NewTable(debug))
}

// debug.traceback([message [, level]])
// If message is non-nil and non-string, it is returned unchanged (Lua 5.4).
func luaDebugTraceback(v *vm.VM) int {
	msg := v.Get(1)
	if !msg.IsNil() && !msg.IsString() {
		v.Set(0, msg)
		return 1
	}

	msgStr := ""
	if !msg.IsNil() {
		msgStr = msg.AsString()
	}

	level := 1
	if !v.Get(2).IsNil() {
		if l, ok := v.Get(2).ToInt(); ok {
			level = int(l)
		}
	}

	v.Set(0, vm.NewString(v.Traceback(msgStr, level)))
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

	if arg1.IsCallable() {
		// debug.getinfo(func [, what])
		info = v.GetFuncInfo(arg1)
		if info == nil {
			v.Set(0, vm.Nil)
			return 1
		}
		what = "flnStu" // default
		if !v.Get(2).IsNil() {
			what = v.Get(2).AsString()
		}
	} else {
		// debug.getinfo(level [, what])
		level, ok := arg1.ToInt()
		if !ok {
			panic("bad argument #1 to 'getinfo' (value expected)")
		}
		if level < 0 {
			v.Set(0, vm.Nil)
			return 1
		}
		// Level 0 = getinfo itself (native frame at top of stack).
		// The VM's GetFrameInfo uses: idx = len(callStack) - 1 - level.
		// Since getinfo's native frame is at the top, level 0 maps directly.
		info = v.GetFrameInfo(int(level))
		if info == nil {
			v.Set(0, vm.Nil)
			return 1
		}
		what = "flnStu" // default
		if !v.Get(2).IsNil() {
			what = v.Get(2).AsString()
		}
	}

	// Validate the what string ('>' is C API only, not valid at Lua level)
	for _, ch := range what {
		if !strings.ContainsRune("flnStuL", ch) {
			panic(fmt.Sprintf("bad argument #2 to 'getinfo' (invalid option '%c')", ch))
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
		case 'L':
			if info.ActiveLines != nil && len(info.ActiveLines) > 0 {
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
		panic("bad argument #1 to 'getupvalue' (function expected)")
	}

	arg2 := v.Get(2)
	idx, ok := arg2.ToInt()
	if !ok {
		panic("bad argument #2 to 'getupvalue' (number expected)")
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
