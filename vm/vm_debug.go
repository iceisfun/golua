package vm

import (
	"fmt"
	"strings"

	"github.com/iceisfun/golua/compiler"
)

// Traceback formats a stack trace string. level is the number of frames to
// skip from the top (0 = current frame, 1 = caller of traceback, etc.).
func (vm *VM) Traceback(msg string, level int) string {
	var b strings.Builder
	if msg != "" {
		b.WriteString(msg)
		b.WriteByte('\n')
	}
	b.WriteString("stack traceback:")

	start := len(vm.callStack) - 1 - level
	if start < 0 {
		start = 0
	}

	for i := start; i >= 0; i-- {
		frame := &vm.callStack[i]
		b.WriteByte('\n')
		b.WriteByte('\t')

		if frame.closure == nil {
			// Native frame
			b.WriteString("[Go]: in ?")
			continue
		}

		if frame.isTailCall {
			b.WriteString("(...tail calls...)")
			continue
		}

		proto := frame.closure.Proto
		source := proto.Source
		if source == "" {
			source = "?"
		}

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
			if i > 0 {
				callerIdx := i - 1
				for callerIdx > 0 && vm.callStack[callerIdx].isTailCall {
					callerIdx--
				}
				if !vm.callStack[callerIdx].isTailCall {
					name, _ = vm.funcNameFromCall(&vm.callStack[callerIdx])
				}
			}
			if name == "" {
				name = vm.frameFuncName(frame)
			}
			fmt.Fprintf(&b, "%s:%d: in function '%s'", source, line, name)
		}
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
		return fmt.Sprintf("<%s:%d>", proto.Source, proto.LineDef)
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
	source = proto.Source
	if source == "" {
		source = "?"
	}

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

	if frame.closure == nil {
		// Native (C) function frame
		info.Source = "=[C]"
		info.ShortSrc = "[C]"
		info.LineDefined = -1
		info.LastLineDefined = -1
		info.CurrentLine = -1
		info.What = "C"
		info.IsVarArg = true
		// Native functions don't have inspectable upvalues/params
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

	// Name inference: look at the caller frame's bytecode
	callerIdx := idx - 1
	if callerIdx >= 0 {
		info.Name, info.NameWhat = vm.funcNameFromCall(&vm.callStack[callerIdx])
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

		return info
	}

	return nil
}

// shortSrc produces a short source name from a full source string,
// matching Lua 5.4's luaO_chunkid behavior.
func shortSrc(source string) string {
	if len(source) == 0 {
		return "[string \"?\"]"
	}
	switch source[0] {
	case '=':
		// User-defined short description
		s := source[1:]
		if len(s) > 60 {
			s = s[:60]
		}
		return s
	case '@':
		// File name
		s := source[1:]
		if len(s) > 60 {
			s = "..." + s[len(s)-57:]
		}
		return s
	default:
		// String source — show first line
		s := source
		if idx := strings.IndexByte(s, '\n'); idx >= 0 {
			s = s[:idx]
		}
		if len(s) > 45 {
			s = s[:45]
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
				return proto.Constants[c].SVal, "field"
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
