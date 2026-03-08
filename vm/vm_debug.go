package vm

import (
	"fmt"
	"strings"

	"github.com/iceisfun/golua/compiler"
)

// Traceback formats a stack trace string. level is the number of frames to
// skip from the top (0 = current frame, 1 = caller of traceback, etc.).
// Long traces are truncated: first 10 entries + "..." + last 11 entries
// (matching Lua 5.4's LEVELS1/LEVELS2 constants).
func (vm *VM) Traceback(msg string, level int) string {
	const (
		levels1 = 10 // number of entries for first part of trace
		levels2 = 11 // number of entries for last part of trace
	)

	var b strings.Builder
	if msg != "" {
		b.WriteString(msg)
		b.WriteByte('\n')
	}
	b.WriteString("stack traceback:")

	start := len(vm.callStack) - 1 - level
	if start < 0 {
		return b.String()
	}

	// Count total frames
	totalFrames := start + 1

	// Determine if we need truncation
	needTruncate := totalFrames > levels1+levels2

	written := 0
	for i := start; i >= 0; i-- {
		// Handle truncation: skip the middle frames
		if needTruncate && written == levels1 {
			// Skip to the last levels2 frames
			remaining := i + 1 // frames from index i down to 0
			if remaining > levels2 {
				b.WriteString("\n\t...")
				i = levels2 // after i-- in for loop, becomes levels2-1
				continue
			}
		}

		frame := &vm.callStack[i]
		b.WriteByte('\n')
		b.WriteByte('\t')

		if frame.closure == nil {
			// Native frame
			b.WriteString("[Go]: in ?")
			written++
			continue
		}

		if frame.isTailCall {
			b.WriteString("(...tail calls...)")
			written++
			continue
		}

		proto := frame.closure.Proto
		source := shortSrc(proto.Source)

		// Determine line number
		line := 0
		pc := frame.pc - 1 // pc points to next instruction, so current is pc-1
		if pc >= 0 && pc < len(proto.Lines) {
			line = proto.Lines[pc]
		}

		// Determine function name
		if i == 0 && proto.LineDef == 0 {
			fmt.Fprintf(&b, "%s:%d: in main chunk", source, line)
		} else {
			// Try to get the function name from the caller's call site
			name := ""
			nameWhat := ""
			if i > 0 {
				callerIdx := i - 1
				for callerIdx > 0 && vm.callStack[callerIdx].isTailCall {
					callerIdx--
				}
				if !vm.callStack[callerIdx].isTailCall {
					name, nameWhat = vm.funcNameFromCall(&vm.callStack[callerIdx])
				}
			}
			// Check for override name (e.g., "close" for __close metamethod calls)
			if name == "" && frame.callName != "" {
				name = frame.callName
				nameWhat = frame.callNameWhat
			}
			if name == "" {
				name = vm.frameFuncName(frame)
			}
			if nameWhat == "metamethod" {
				fmt.Fprintf(&b, "%s:%d: in metamethod '%s'", source, line, name)
			} else {
				fmt.Fprintf(&b, "%s:%d: in function '%s'", source, line, name)
			}
		}
		written++
	}

	return b.String()
}

// frameFuncName attempts to determine a display name for a call frame's function.
func (vm *VM) frameFuncName(frame *callFrame) string {
	if frame.closure == nil {
		return "?"
	}
	proto := frame.closure.Proto
	if proto.Source != "" && proto.LineDef > 0 {
		return fmt.Sprintf("<%s:%d>", shortSrc(proto.Source), proto.LineDef)
	}
	return "?"
}

// StackDepth returns the current call stack depth.
func (vm *VM) StackDepth() int {
	return len(vm.callStack)
}

// Where returns the source file and line number at a given stack level.
// Level 0 = current frame, 1 = caller, etc.
// Returns ("", 0, false) if the level is out of range or the frame is native.
func (vm *VM) Where(level int) (source string, line int, ok bool) {
	idx := len(vm.callStack) - 1 - level
	if idx < 0 || idx >= len(vm.callStack) {
		return "", 0, false
	}

	frame := &vm.callStack[idx]
	if frame.closure == nil {
		return "", 0, false
	}

	proto := frame.closure.Proto
	source = shortSrc(proto.Source)

	pc := frame.pc - 1
	if pc >= 0 && pc < len(proto.Lines) {
		line = proto.Lines[pc]
	}

	return source, line, true
}

// FrameInfo holds debug information about a call stack frame.
type FrameInfo struct {
	// S fields
	Source          string // source name
	ShortSrc        string // short source name
	LineDefined     int    // first line of definition
	LastLineDefined int    // last line of definition
	What            string // "Lua", "C", or "main"

	// l fields
	CurrentLine int // current executing line (-1 if not available)

	// n fields
	Name     string // function name (empty if unknown)
	NameWhat string // "global", "local", "method", "field", "upvalue", or ""

	// u fields
	NUps     int  // number of upvalues
	NParams  int  // number of fixed parameters
	IsVarArg bool // true if function is vararg

	// t fields
	IsTailCall bool

	// r fields
	FTransfer int // first "transfer" value index (1-based, for hooks)
	NTransfer int // number of transfer values

	// f field
	Func Value // the function value itself

	// L field
	ActiveLines map[int]bool // set of active lines (nil for C functions)
}

// GetFrameInfo returns debug info for the call frame at the given level.
// Level 0 = current frame, 1 = caller, etc.
// Returns nil if the level is out of range.
func (vm *VM) GetFrameInfo(level int) *FrameInfo {
	idx := len(vm.callStack) - 1 - level
	if idx < 0 || idx >= len(vm.callStack) {
		return nil
	}

	frame := &vm.callStack[idx]
	info := &FrameInfo{}

	// Transfer info (from hooks, set on frame before hook fires)
	info.FTransfer = frame.ftransfer
	info.NTransfer = frame.ntransfer

	if frame.closure == nil {
		// Native (C) function frame
		info.Source = "=[C]"
		info.ShortSrc = "[C]"
		info.LineDefined = -1
		info.LastLineDefined = -1
		info.CurrentLine = -1
		info.What = "C"
		info.IsVarArg = true
		// Name inference: look at the caller frame's bytecode
		callerIdx := idx - 1
		if callerIdx >= 0 {
			info.Name, info.NameWhat = vm.funcNameFromCall(&vm.callStack[callerIdx])
		}
		return info
	}

	proto := frame.closure.Proto
	info.Source = proto.Source
	info.ShortSrc = shortSrc(proto.Source)
	info.LineDefined = proto.LineDef
	info.LastLineDefined = proto.LastLine
	info.NUps = len(proto.Upvalues)
	info.NParams = proto.NumParams
	info.IsVarArg = proto.IsVarArg
	info.IsTailCall = frame.isTailCall
	info.Func = NewFunction(frame.closure)

	if proto.LineDef == 0 {
		info.What = "main"
	} else {
		info.What = "Lua"
	}

	// Current line
	pc := frame.pc - 1
	if pc >= 0 && pc < len(proto.Lines) {
		info.CurrentLine = proto.Lines[pc]
	} else {
		info.CurrentLine = -1
	}

	// Active lines
	info.ActiveLines = make(map[int]bool)
	for _, line := range proto.Lines {
		if line > 0 {
			info.ActiveLines[line] = true
		}
	}
	// Include the 'end' line (LastLine) which may not have an instruction,
	// but only when the function has line info (non-stripped).
	if proto.LastLine > 0 && len(proto.Lines) > 0 {
		info.ActiveLines[proto.LastLine] = true
	}

	// Name inference: look at the caller frame's bytecode
	callerIdx := idx - 1
	if callerIdx >= 0 {
		info.Name, info.NameWhat = vm.funcNameFromCall(&vm.callStack[callerIdx])
	}

	// If bytecode-based name inference failed, use the frame's override name
	// (e.g., "close" for __close metamethod calls)
	if info.NameWhat == "" && frame.callName != "" {
		info.Name = frame.callName
		info.NameWhat = frame.callNameWhat
	}

	// When inside a hook and name couldn't be inferred from caller,
	// mark as "hook". This happens because the hook function is called
	// by fireHook/ProtectedCall, not by a CALL instruction in the caller.
	if vm.inHook && info.NameWhat == "" {
		info.NameWhat = "hook"
	}

	return info
}

// GetFuncInfo returns debug info for a function value (not a stack frame).
// Stack-related fields (CurrentLine, IsTailCall, Name) are not available.
func (vm *VM) GetFuncInfo(fn Value) *FrameInfo {
	info := &FrameInfo{
		CurrentLine: -1,
	}

	if fn.IsNativeFunc() {
		info.Source = "=[C]"
		info.ShortSrc = "[C]"
		info.LineDefined = -1
		info.LastLineDefined = -1
		info.What = "C"
		info.IsVarArg = true
		info.Func = fn
		return info
	}

	if fn.IsFunction() {
		closure := fn.AsClosure()
		proto := closure.Proto
		info.Source = proto.Source
		info.ShortSrc = shortSrc(proto.Source)
		info.LineDefined = proto.LineDef
		info.LastLineDefined = proto.LastLine
		info.NUps = len(closure.Upvalues)
		info.NParams = proto.NumParams
		info.IsVarArg = proto.IsVarArg
		info.Func = fn

		if proto.LineDef == 0 {
			info.What = "main"
		} else {
			info.What = "Lua"
		}

		// Active lines
		info.ActiveLines = make(map[int]bool)
		for _, line := range proto.Lines {
			if line > 0 {
				info.ActiveLines[line] = true
			}
		}
		// Include the 'end' line (LastLine) which may not have an instruction,
		// but only when the function has line info (non-stripped).
		if proto.LastLine > 0 && len(proto.Lines) > 0 {
			info.ActiveLines[proto.LastLine] = true
		}

		return info
	}

	return nil
}

// shortSrc produces a short source name from a full source string,
// matching Lua 5.4's luaO_chunkid behavior.
func shortSrc(source string) string {
	if len(source) == 0 {
		return "[string \"\"]"
	}
	switch source[0] {
	case '=':
		// User-defined short description (Lua 5.4 LUA_IDSIZE=60, minus null = 59)
		s := source[1:]
		if len(s) > 59 {
			s = s[:59]
		}
		return s
	case '@':
		// File name (max 59 visible chars; longer names get "..." prefix)
		s := source[1:]
		if len(s) >= 60 {
			s = "..." + s[len(s)-56:]
		}
		return s
	default:
		// String source — show first line, add "..." if truncated
		s := source
		truncated := false
		// Null byte truncates the name, matching Lua 5.4's C string behavior
		// where the null byte naturally terminates the C string. This is not
		// treated as "truncation" for the purposes of adding "..." suffix.
		if idx := strings.IndexByte(s, '\x00'); idx >= 0 {
			s = s[:idx]
		}
		if idx := strings.IndexByte(s, '\n'); idx >= 0 {
			s = s[:idx]
			truncated = true
		}
		if len(s) >= 45 {
			s = s[:45]
			truncated = true
		}
		if truncated {
			return "[string \"" + s + "...\"]"
		}
		return "[string \"" + s + "\"]"
	}
}

// funcNameFromCall inspects the caller frame's bytecode to infer the name
// of the function being called. It looks at the instruction at the caller's
// current PC (which should be a CALL or TAILCALL) and traces back to find
// what loaded the function register.
func (vm *VM) funcNameFromCall(callerFrame *callFrame) (name, nameWhat string) {
	if callerFrame.closure == nil {
		return "", ""
	}
	proto := callerFrame.closure.Proto

	// The caller's pc points to the next instruction to execute.
	// The CALL/TAILCALL instruction that triggered the call is at pc-1.
	pc := callerFrame.pc - 1
	if pc < 0 || pc >= len(proto.Code) {
		return "", ""
	}

	inst := proto.Code[pc]
	op := inst.OpCode()

	// For-in iterator call (OP_TFORCALL): name is "for iterator"
	if op == compiler.OP_TFORCALL {
		return "for iterator", "for iterator"
	}

	// Metamethod calls: detect arithmetic/comparison/index metamethods.
	// In GoLua, metamethods are called from within the arith/index handler,
	// so pc-1 points to the triggering instruction. For arithmetic ops, the
	// next instruction (at pc+1) is OP_MMBIN/MMBINI/MMBINK with the tag.
	if mmName := vm.metamethodNameFromOp(proto, pc); mmName != "" {
		return mmName, "metamethod"
	}

	if op != compiler.OP_CALL && op != compiler.OP_TAILCALL {
		return "", ""
	}

	reg := inst.A() // register holding the function

	// Scan backwards for the instruction that wrote to register `reg`.
	for i := pc - 1; i >= 0; i-- {
		prev := proto.Code[i]
		prevOp := prev.OpCode()
		prevA := prev.A()

		// Skip instructions that don't write to our target register.
		if prevA != reg {
			// Some opcodes write to A+1 (e.g., SELF writes R[A+1] = R[B]).
			// We only care about the function register (A of the CALL).
			continue
		}

		switch prevOp {
		case compiler.OP_GETTABUP:
			// R[A] := UpValue[B][K[C]:string]
			c := prev.C()
			if c < len(proto.Constants) && proto.Constants[c].Type == compiler.ValString {
				if prev.B() == 0 {
					// Upvalue 0 is _ENV — this is a global access
					return proto.Constants[c].SVal, "global"
				}
				return proto.Constants[c].SVal, "upvalue"
			}
			return "", ""
		case compiler.OP_GETFIELD:
			// R[A] := R[B][K[C]:string]
			c := prev.C()
			if c < len(proto.Constants) && proto.Constants[c].Type == compiler.ValString {
				what := "field"
				if localName(proto, prev.B(), i) == "_ENV" || isUpvalEnv(proto, i, prev.B()) {
					what = "global"
				}
				return proto.Constants[c].SVal, what
			}
			return "", ""
		case compiler.OP_SELF:
			// R[A+1] := R[B]; R[A] := R[B][K[C]:string]
			c := prev.C()
			if c < len(proto.Constants) && proto.Constants[c].Type == compiler.ValString {
				return proto.Constants[c].SVal, "method"
			}
			return "", ""
		case compiler.OP_MOVE:
			// R[A] := R[B] — try to get the local variable name
			name := localName(proto, prev.B(), i)
			if name != "?" {
				return name, "local"
			}
			return "", ""
		case compiler.OP_GETUPVAL:
			// R[A] := UpValue[B]
			idx := prev.B()
			if idx < len(proto.Upvalues) {
				return proto.Upvalues[idx].Name, "upvalue"
			}
			return "", ""
		default:
			// Found an instruction writing to reg, but can't infer a name.
			return "", ""
		}
	}

	return "", ""
}

// metamethodNameFromOp checks whether the instruction at pc in proto is an
// opcode that invokes a metamethod (arithmetic, comparison, index, unary, etc.).
// Returns the metamethod name without the "__" prefix, or "" if not a metamethod call.
func (vm *VM) metamethodNameFromOp(proto *compiler.Proto, pc int) string {
	if pc < 0 || pc >= len(proto.Code) {
		return ""
	}
	inst := proto.Code[pc]
	op := inst.OpCode()

	switch op {
	// Arithmetic/bitwise ops: the next instruction should be OP_MMBIN/MMBINI/MMBINK
	// containing the metamethod tag in its C field.
	case compiler.OP_ADD, compiler.OP_SUB, compiler.OP_MUL, compiler.OP_MOD,
		compiler.OP_POW, compiler.OP_DIV, compiler.OP_IDIV,
		compiler.OP_BAND, compiler.OP_BOR, compiler.OP_BXOR,
		compiler.OP_SHL, compiler.OP_SHR:
		// Look at next instruction for MMBIN with tag
		if pc+1 < len(proto.Code) {
			next := proto.Code[pc+1]
			nextOp := next.OpCode()
			if nextOp == compiler.OP_MMBIN || nextOp == compiler.OP_MMBINI || nextOp == compiler.OP_MMBINK {
				tag := compiler.MetamethodTag(next.C())
				name := tag.String()
				if len(name) > 2 && name[:2] == "__" {
					return name[2:]
				}
				return name
			}
		}
		return ""

	case compiler.OP_ADDI:
		if pc+1 < len(proto.Code) {
			next := proto.Code[pc+1]
			if next.OpCode() == compiler.OP_MMBINI {
				tag := compiler.MetamethodTag(next.C())
				name := tag.String()
				if len(name) > 2 && name[:2] == "__" {
					return name[2:]
				}
			}
		}
		return ""

	case compiler.OP_ADDK, compiler.OP_SUBK, compiler.OP_MULK, compiler.OP_MODK,
		compiler.OP_POWK, compiler.OP_DIVK, compiler.OP_IDIVK,
		compiler.OP_BANDK, compiler.OP_BORK, compiler.OP_BXORK:
		if pc+1 < len(proto.Code) {
			next := proto.Code[pc+1]
			if next.OpCode() == compiler.OP_MMBINK {
				tag := compiler.MetamethodTag(next.C())
				name := tag.String()
				if len(name) > 2 && name[:2] == "__" {
					return name[2:]
				}
			}
		}
		return ""

	case compiler.OP_SHLI, compiler.OP_SHRI:
		if pc+1 < len(proto.Code) {
			next := proto.Code[pc+1]
			if next.OpCode() == compiler.OP_MMBINI {
				tag := compiler.MetamethodTag(next.C())
				name := tag.String()
				if len(name) > 2 && name[:2] == "__" {
					return name[2:]
				}
			}
		}
		return ""

	// Unary operations
	case compiler.OP_UNM:
		return "unm"
	case compiler.OP_BNOT:
		return "bnot"
	case compiler.OP_LEN:
		return "len"

	// String concatenation
	case compiler.OP_CONCAT:
		return "concat"

	// Comparison operations
	case compiler.OP_EQ, compiler.OP_EQK, compiler.OP_EQI:
		return "eq"
	case compiler.OP_LT, compiler.OP_LTI:
		return "lt"
	case compiler.OP_LE, compiler.OP_LEI:
		return "le"
	case compiler.OP_GTI:
		return "lt" // __lt with reversed operands
	case compiler.OP_GEI:
		return "le" // __le with reversed operands

	// Index operations (when __index metamethod is called)
	case compiler.OP_GETTABLE, compiler.OP_GETI, compiler.OP_GETFIELD:
		return "index"
	case compiler.OP_SETTABLE, compiler.OP_SETI, compiler.OP_SETFIELD:
		return "newindex"

	default:
		return ""
	}
}

// GetLocal returns the name and value of local variable #index at the given
// stack level. Level 0 = current frame, 1 = caller, etc.
// index is 1-based. Negative indices access varargs: -1 = first vararg, etc.
// Returns ("", Nil, false) if out of range.
func (vm *VM) GetLocal(level, index int) (string, Value, bool) {
	idx := len(vm.callStack) - 1 - level
	if idx < 0 || idx >= len(vm.callStack) {
		return "", Nil, false
	}

	frame := &vm.callStack[idx]
	if frame.closure == nil {
		// Native (C) function frame: access stack slots directly.
		// Stack layout: base+0 = function value, base+1 = first arg, ...
		// getlocal index 1 = base+1 (first arg), matching Lua 5.4.
		if index <= 0 {
			return "", Nil, false
		}
		stackIdx := frame.base + index
		if stackIdx < 0 || stackIdx >= len(vm.stack) {
			return "", Nil, false
		}
		// Upper bound: next frame's base (if any) or vm.top
		limit := vm.top
		if idx+1 < len(vm.callStack) {
			limit = vm.callStack[idx+1].base
		}
		if stackIdx >= limit {
			return "", Nil, false
		}
		return "(C temporary)", vm.stack[stackIdx], true
	}

	// Negative index: access varargs
	if index < 0 {
		if !frame.isVararg || frame.numVararg == 0 {
			return "", Nil, false
		}
		varIdx := -index - 1 // 0-based vararg index
		if varIdx >= frame.numVararg {
			return "", Nil, false
		}
		stackIdx := frame.varargPos + varIdx
		val := Nil
		if stackIdx >= 0 && stackIdx < len(vm.stack) {
			val = vm.stack[stackIdx]
		}
		return "(vararg)", val, true
	}

	proto := frame.closure.Proto
	pc := frame.pc - 1
	if pc < 0 {
		pc = 0
	}

	// The index-th local variable maps to register (index-1).
	// Use localName to resolve the name from the register number.
	reg := index - 1
	name := localName(proto, reg, pc)
	if name == "?" {
		return "", Nil, false
	}

	stackIdx := frame.base + reg
	val := Nil
	if stackIdx >= 0 && stackIdx < len(vm.stack) {
		val = vm.stack[stackIdx]
	}

	return name, val, true
}

// GetFuncLocal returns the name of local variable #index from a function's
// prototype (without needing a live stack frame). Only parameter names are
// available. index is 1-based. Returns ("", false) if out of range.
func (vm *VM) GetFuncLocal(fn Value, index int) (string, bool) {
	if fn.IsNativeFunc() || !fn.IsFunction() {
		return "", false
	}
	closure := fn.AsClosure()
	if closure == nil {
		return "", false
	}
	proto := closure.Proto
	// For a function not on the stack, we check locals at PC 0
	// Only parameters (StartPC == 0) are visible
	reg := index - 1
	name := localName(proto, reg, 0)
	if name == "?" {
		// For stripped functions (no debug info), registers within the
		// function's stack frame are reported as "(temporary)" — matching
		// Lua 5.4 behavior for locals without names.
		if len(proto.Locals) == 0 && index >= 1 && index <= proto.NumParams {
			return "(temporary)", true
		}
		return "", false
	}
	return name, true
}

// SetLocal sets the value of local variable #index at the given stack level.
// Returns the name of the variable, or ("", false) if out of range.
func (vm *VM) SetLocal(level, index int, val Value) (string, bool) {
	idx := len(vm.callStack) - 1 - level
	if idx < 0 || idx >= len(vm.callStack) {
		return "", false
	}

	frame := &vm.callStack[idx]
	if frame.closure == nil {
		if index <= 0 {
			return "", false
		}
		stackIdx := frame.base + index
		if stackIdx < 0 || stackIdx >= len(vm.stack) {
			return "", false
		}
		limit := vm.top
		if idx+1 < len(vm.callStack) {
			limit = vm.callStack[idx+1].base
		}
		if stackIdx >= limit {
			return "", false
		}
		vm.stack[stackIdx] = val
		return "(C temporary)", true
	}

	// Negative index: access varargs
	if index < 0 {
		if !frame.isVararg || frame.numVararg == 0 {
			return "", false
		}
		varIdx := -index - 1 // 0-based vararg index
		if varIdx >= frame.numVararg {
			return "", false
		}
		stackIdx := frame.varargPos + varIdx
		if stackIdx >= 0 && stackIdx < len(vm.stack) {
			vm.stack[stackIdx] = val
		}
		return "(vararg)", true
	}

	proto := frame.closure.Proto
	pc := frame.pc - 1
	if pc < 0 {
		pc = 0
	}

	reg := index - 1
	name := localName(proto, reg, pc)
	if name == "?" {
		return "", false
	}

	stackIdx := frame.base + reg
	if stackIdx >= 0 && stackIdx < len(vm.stack) {
		vm.stack[stackIdx] = val
	}

	return name, true
}

// IsValidLevel checks if a stack level is within the current call stack bounds.
func (vm *VM) IsValidLevel(level int) bool {
	idx := len(vm.callStack) - 1 - level
	return idx >= 0 && idx < len(vm.callStack)
}

// GetRegistry returns the VM's registry table, creating it on first access.
func (vm *VM) GetRegistry() LuaTable {
	if vm.registry == nil {
		vm.registry = NewEmptyTable()
	}
	return vm.registry
}

// regObjName returns a descriptive string for the value in register `reg`
// at the given PC within a prototype, by scanning backwards through bytecode.
// Returns (name, what) where what is "field", "global", "local", "upvalue",
// or "" if unknown. This matches Lua 5.4's getobjname / kname behavior.
func regObjName(proto *compiler.Proto, pc int, reg int) (string, string) {
	// First check if the register is a local variable at the faulting PC.
	// This handles cases like "local a; a(13)" where the LOADNIL that
	// initialises register 0 occurs before the local's StartPC.
	name := localName(proto, reg, pc)
	if name != "" && name != "?" && !isInternalName(name) {
		return name, "local"
	}

	for i := pc - 1; i >= 0; i-- {
		inst := proto.Code[i]
		op := inst.OpCode()
		a := inst.A()
		if a != reg {
			continue
		}
		switch op {
		case compiler.OP_LOADK:
			bx := inst.Bx()
			if bx < len(proto.Constants) && proto.Constants[bx].Type == compiler.ValString {
				return proto.Constants[bx].SVal, "constant"
			}
			return "", ""
		case compiler.OP_LOADKX:
			if i+1 < len(proto.Code) {
				ax := proto.Code[i+1].Ax()
				if ax < len(proto.Constants) && proto.Constants[ax].Type == compiler.ValString {
					return proto.Constants[ax].SVal, "constant"
				}
			}
			return "", ""
		case compiler.OP_GETFIELD:
			c := inst.C()
			if c < len(proto.Constants) && proto.Constants[c].Type == compiler.ValString {
				what := "field"
				if localName(proto, inst.B(), i) == "_ENV" || isUpvalEnv(proto, i, inst.B()) {
					what = "global"
				}
				return proto.Constants[c].SVal, what
			}
			return "", ""
		case compiler.OP_GETTABUP:
			c := inst.C()
			if c < len(proto.Constants) && proto.Constants[c].Type == compiler.ValString {
				if inst.B() == 0 {
					return proto.Constants[c].SVal, "global"
				}
				return proto.Constants[c].SVal, "field"
			}
			return "", ""
		case compiler.OP_MOVE:
			reg = inst.B()
			// After following OP_MOVE, check if the new register is a local.
			ln := localName(proto, reg, i)
			if ln != "" && ln != "?" && !isInternalName(ln) {
				return ln, "local"
			}
			continue
		case compiler.OP_GETUPVAL:
			b := inst.B()
			if b < len(proto.Upvalues) && proto.Upvalues[b].Name != "" {
				return proto.Upvalues[b].Name, "upvalue"
			}
			return "", ""
		case compiler.OP_SELF:
			c := inst.C()
			if c < len(proto.Constants) && proto.Constants[c].Type == compiler.ValString {
				return proto.Constants[c].SVal, "method"
			}
			return "", ""
		case compiler.OP_GETI:
			return "integer index", "field"
		case compiler.OP_GETTABLE:
			// R[A] = R[B][R[C]] — try to resolve R[C] to a constant name.
			// When the table was loaded via GETUPVAL of _ENV (upvalue 0),
			// report "global" instead of "field". This handles the fallback
			// path emitGetTabUp uses when constant index > MaxArgC.
			b := inst.B()
			c := inst.C()
			kn := kName(proto, i, c)
			if kn != "" {
				what := "field"
				if isUpvalEnv(proto, i, b) {
					what = "global"
				}
				return kn, what
			}
			return "", ""
		default:
			ln := localName(proto, reg, i)
			if ln != "" && ln != "?" && !isInternalName(ln) {
				return ln, "local"
			}
			return "", ""
		}
	}
	return "", ""
}

// isInternalName returns true if the name is a compiler-generated internal
// name (starts with '(' such as "(for state)" or "(for control)").
func isInternalName(name string) bool {
	return len(name) > 0 && name[0] == '('
}

// isUpvalEnv checks whether the register at the given PC was loaded via
// GETUPVAL from upvalue 0 (_ENV). Used to distinguish "global" from "field"
// when GETTABLE is the fallback for GETTABUP with large constant indices.
func isUpvalEnv(proto *compiler.Proto, pc int, reg int) bool {
	for i := pc - 1; i >= 0; i-- {
		inst := proto.Code[i]
		if inst.A() != reg {
			continue
		}
		if inst.OpCode() == compiler.OP_GETUPVAL && inst.B() == 0 {
			return true
		}
		return false
	}
	return false
}

// kName resolves a register reference to a constant name at the given PC.
// If register reg was loaded from a constant (via OP_LOADK/OP_LOADKX) and
// the constant is a string, returns that string. Otherwise returns "".
func kName(proto *compiler.Proto, pc int, reg int) string {
	for i := pc - 1; i >= 0; i-- {
		inst := proto.Code[i]
		op := inst.OpCode()
		a := inst.A()
		if a != reg {
			continue
		}
		switch op {
		case compiler.OP_LOADK:
			bx := inst.Bx()
			if bx < len(proto.Constants) && proto.Constants[bx].Type == compiler.ValString {
				return proto.Constants[bx].SVal
			}
			return ""
		case compiler.OP_LOADKX:
			if i+1 < len(proto.Code) {
				ax := proto.Code[i+1].Ax()
				if ax < len(proto.Constants) && proto.Constants[ax].Type == compiler.ValString {
					return proto.Constants[ax].SVal
				}
			}
			return ""
		case compiler.OP_MOVE:
			reg = inst.B()
			continue
		default:
			return ""
		}
	}
	return ""
}
