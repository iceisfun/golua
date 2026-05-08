package vm

import (
	"fmt"
	"strings"

	"github.com/iceisfun/golua/v2/compiler"
)

// terminalCFuncVal is a singleton native function value representing the
// synthetic [C] entry point at the bottom of the main VM call stack.
// Lua 5.4 exposes this via debug.getinfo's "f" option on the outermost level.
var terminalCFuncVal = NewNativeFunc(func(v *VM) int { return 0 })

// terminalCFunc returns the synthetic [C] function value for the main VM's
// outermost stack frame.
func (vm *VM) terminalCFunc() Value {
	return terminalCFuncVal
}

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

	if level < 0 {
		return b.String()
	}
	start := len(vm.callStack) - 1 - level
	if start < 0 {
		return b.String()
	}

	// Count total frames. The main VM has a synthetic final [C] frame;
	// coroutines do not.
	totalFrames := start + 1
	if vm.yieldCh == nil {
		totalFrames++
	}

	// Determine if we need truncation
	needTruncate := totalFrames > levels1+levels2

	written := 0
	for i := start; i >= 0; i-- {
		// Handle truncation: skip the middle frames
		if needTruncate && written == levels1 {
			// Skip to the last tail frames. For the main VM, the synthetic
			// [C]: in ? frame occupies the final slot; coroutines use all slots
			// for real frames.
			tailSlots := levels2 - 1
			if vm.yieldCh != nil {
				tailSlots = levels2
			}
			remaining := i + 1 // frames from index i down to 0
			if remaining > tailSlots {
				skipped := remaining - tailSlots
				fmt.Fprintf(&b, "\n\t...\t(skipping %d levels)", skipped)
				i = tailSlots // after i-- in for loop, becomes tailSlots-1
				continue
			}
		}

		frame := &vm.callStack[i]
		b.WriteByte('\n')
		b.WriteByte('\t')

		if frame.closure == nil {
			name := ""
			nameWhat := ""
			if i > 0 {
				name, nameWhat = vm.funcNameFromCall(&vm.callStack[i-1])
			}
			if name == "" && frame.callName != "" && !suppressLuaFrameCallName(frame, hasHigherLuaFrame(vm.callStack, i, start)) {
				name = frame.callName
				nameWhat = frame.callNameWhat
			}
			name = vm.tracebackNativeName(frame, name, nameWhat)
			switch {
			case nameWhat == "metamethod":
				fmt.Fprintf(&b, "[C]: in metamethod '%s'", name)
			case name != "" && (nameWhat == "global" || nameWhat == "local" || nameWhat == "field" || nameWhat == "upvalue" || nameWhat == "method"):
				fmt.Fprintf(&b, "[C]: in %s '%s'", nameWhat, name)
			case name != "":
				fmt.Fprintf(&b, "[C]: in function '%s'", name)
			default:
				b.WriteString("[C]: in ?")
			}
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
		} else if pc < 0 && len(proto.Lines) > 0 {
			line = proto.Lines[0]
		}

		// Determine function name
		if i == 0 && proto.LineDef == 0 {
			fmt.Fprintf(&b, "%s:%d: in main chunk", source, line)
		} else {
			// Try to get the function name from the caller's call site
			name := ""
			nameWhat := ""
			// Hook functions are called by the VM, not by a CALL instruction.
			// Skip caller-based name resolution and use "hook" directly.
			if vm.inHook && frame.funcValue.RawEqual(vm.hookFunc) {
				name = "?"
				nameWhat = "hook"
			} else {
				suppressCallName := suppressLuaFrameCallName(frame, hasHigherLuaFrame(vm.callStack, i, start))
				if i > 0 && !frame.isTailCall {
					callerIdx := i - 1
					for callerIdx > 0 && vm.callStack[callerIdx].isTailCall {
						callerIdx--
					}
					if !vm.callStack[callerIdx].isTailCall {
						name, nameWhat = vm.funcNameFromCall(&vm.callStack[callerIdx])
					}
				}
				if suppressCallName && nameWhat == "metamethod" && name == "close" {
					name = ""
					nameWhat = ""
				}
				// Check for override name (e.g., "close" for __close metamethod calls)
				if name == "" && frame.callName != "" && !suppressCallName {
					name = frame.callName
					nameWhat = frame.callNameWhat
				}
				if suppressCallName {
					name = ""
					nameWhat = ""
				}
			}
			// If no name yet, try reverse lookup in globals for Lua functions
			if name == "" && frame.closure != nil {
				if resolved, ok := vm.lookupNativeFuncName(frame.funcValue); ok {
					name = resolved
				}
			}
			if name == "" {
				name = vm.frameFuncName(frame)
			}
			if nameWhat == "metamethod" {
				fmt.Fprintf(&b, "%s:%d: in metamethod '%s'", source, line, name)
			} else if nameWhat == "hook" {
				fmt.Fprintf(&b, "%s:%d: in hook '%s'", source, line, name)
			} else if len(name) > 0 && name[0] == '<' {
				// Anonymous function: "in function <file:line>" (no quotes)
				fmt.Fprintf(&b, "%s:%d: in function %s", source, line, name)
			} else if nameWhat == "global" || nameWhat == "local" || nameWhat == "field" || nameWhat == "upvalue" || nameWhat == "method" {
				fmt.Fprintf(&b, "%s:%d: in %s '%s'", source, line, nameWhat, name)
			} else {
				fmt.Fprintf(&b, "%s:%d: in function '%s'", source, line, name)
			}
		}
		written++

		// After printing a tail-called frame, add the tail calls marker
		// to indicate that intermediate tail call frames were elided.
		if frame.isTailCall {
			b.WriteString("\n\t(...tail calls...)")
			written++
		}
	}

	// Only append [C]: in ? for the main VM, not coroutines.
	if vm.yieldCh == nil {
		b.WriteString("\n\t[C]: in ?")
	}

	return b.String()
}

func (vm *VM) tracebackNativeName(frame *callFrame, name, nameWhat string) string {
	if name == "" || name == "?" {
		if resolved, ok := vm.lookupNativeFuncName(frame.funcValue); ok {
			return resolved
		}
		return name
	}
	// Lua 5.5: keep "field" names unqualified (e.g. "format", not "string.format")
	// to match the reference traceback "[C]: in field 'format'".
	return name
}

func (vm *VM) lookupNativeFuncName(fn Value) (string, bool) {
	if fn.IsNil() || vm.globals == nil {
		return "", false
	}
	if name, ok := lookupFuncNameInTable(vm.globals, fn, "", 2); ok {
		return name, true
	}
	return "", false
}

func lookupFuncNameInTable(tbl LuaTable, target Value, prefix string, depth int) (string, bool) {
	if tbl == nil || depth < 0 {
		return "", false
	}
	for k, v, err := tbl.Next(Nil); err == nil && !k.IsNil(); k, v, err = tbl.Next(k) {
		if !k.IsString() {
			continue
		}
		name := k.AsString()
		full := name
		if prefix != "" {
			full = prefix + "." + name
		}
		if v.RawEqual(target) {
			full = strings.TrimPrefix(full, "_G.")
			return full, true
		}
		if depth > 0 && v.IsTable() {
			if prefix == "" && name == "_G" {
				continue
			}
			if nested, ok := lookupFuncNameInTable(v.AsTable(), target, full, depth-1); ok {
				return nested, true
			}
		}
	}
	return "", false
}

// TracebackFromLastError formats traceback using the most recently captured
// error stack (if available), then consumes that snapshot (unless the VM's
// call stack is empty, indicating a dead coroutine where repeat calls are expected).
func (vm *VM) TracebackFromLastError(msg string, level int) string {
	if len(vm.lastErrorCallStack) == 0 {
		return vm.Traceback(msg, level)
	}
	saved := vm.callStack
	vm.callStack = vm.lastErrorCallStack
	out := vm.Traceback(msg, level)
	vm.callStack = saved
	// Only consume the snapshot if the VM still has an active call stack.
	// Dead coroutines (empty callStack) preserve the snapshot for repeated
	// debug.traceback calls.
	if len(saved) > 0 {
		vm.lastErrorCallStack = nil
	}
	return out
}

// ClearLastErrorCallStack explicitly clears the saved error call stack snapshot,
// allowing a subsequent ProtectedCall to capture a fresh snapshot. Used by the
// CLI error reporter after consuming the outer traceback, before calling
// __tostring which may trigger an inner error with its own traceback.
func (vm *VM) ClearLastErrorCallStack() {
	vm.lastErrorCallStack = nil
}

// HasLastErrorTraceback reports whether an error stack snapshot is available.
func (vm *VM) HasLastErrorTraceback() bool {
	return len(vm.lastErrorCallStack) > 0
}

// SaveLastErrorCallStack returns (and clears) the current lastErrorCallStack
// so it can be restored later if a nested ProtectedCall succeeds.
func (vm *VM) SaveLastErrorCallStack() []callFrame {
	saved := vm.lastErrorCallStack
	vm.lastErrorCallStack = nil
	return saved
}

// RestoreLastErrorCallStack restores a previously saved error call stack.
func (vm *VM) RestoreLastErrorCallStack(saved []callFrame) {
	vm.lastErrorCallStack = saved
}

func (vm *VM) debugCallStack() ([]callFrame, bool) {
	if len(vm.callStack) > 0 {
		return vm.callStack, true
	}
	return vm.lastErrorCallStack, false
}

func suppressLuaFrameCallName(frame *callFrame, hasHigherLuaFrame bool) bool {
	if frame == nil || frame.closure == nil {
		return false
	}
	if frame.suppressTracebackName {
		return true
	}
	if frame.callNameWhat != "metamethod" || frame.callName != "close" {
		return false
	}
	return hasHigherLuaFrame
}

func hasHigherLuaFrame(stack []callFrame, idx, top int) bool {
	for i := top; i > idx; i-- {
		if stack[i].closure != nil {
			return true
		}
	}
	return false
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
	} else if pc < 0 && len(proto.Lines) > 0 {
		line = proto.Lines[0]
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
	ExtraArgs  int // number of extra arguments from __call metamethod chains

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
	stack, active := vm.debugCallStack()
	idx := len(stack) - 1 - level
	if idx < 0 || idx >= len(stack) {
		// For the main VM (not coroutines), synthesize a terminal [C] frame
		// at the level just past the real stack, matching Lua 5.4's C runtime frame.
		if vm.yieldCh == nil && idx == -1 {
			return &FrameInfo{
				Source:          "=[C]",
				ShortSrc:        "[C]",
				LineDefined:     -1,
				LastLineDefined: -1,
				CurrentLine:     -1,
				What:            "C",
				IsVarArg:        true,
				Func:            vm.terminalCFunc(),
			}
		}
		return nil
	}

	frame := &stack[idx]
	info := &FrameInfo{}

	// Transfer info is only observable during hook execution.
	// Outside hooks, Lua 5.4 reports 0/0 for these fields.
	if vm.inHook {
		info.FTransfer = frame.ftransfer
		info.NTransfer = frame.ntransfer
	}

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
			info.Name, info.NameWhat = vm.funcNameFromCall(&stack[callerIdx])
		}
		info.ExtraArgs = frame.extraArgs
		info.Func = frame.funcValue
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
	info.ExtraArgs = frame.extraArgs
	info.Func = frame.funcValue

	if proto.LineDef == 0 {
		info.What = "main"
	} else {
		info.What = "Lua"
	}

	// Current line
	pc := frame.pc - 1
	if pc >= 0 && pc < len(proto.Lines) {
		info.CurrentLine = proto.Lines[pc]
	} else if pc < 0 && len(proto.Lines) > 0 {
		// pc == -1 means frame.pc == 0: function just entered (e.g., call hook).
		// Use the first instruction's line number.
		info.CurrentLine = proto.Lines[0]
	} else {
		info.CurrentLine = -1
	}

	info.ActiveLines = activeLines(proto)

	// When inside a hook, the hook function was called by the VM's hook
	// mechanism, not by a CALL instruction. The caller frame's bytecode
	// at the current PC is the hooked instruction, not a CALL that invoked
	// the hook. Looking up a name from that bytecode would produce wrong
	// results (e.g., "metamethod" for GETTABUP near an MMBIN).
	// Check this FIRST, before attempting caller-based name resolution.
	if active && vm.inHook && info.Func.RawEqual(vm.hookFunc) {
		info.Name = "?"
		info.NameWhat = "hook"
	} else if !frame.isTailCall {
		// Name inference: look at the caller frame's bytecode.
		// For tail calls, the original caller frame is gone, so name resolution
		// must fail (returning empty name/namewhat), matching Lua 5.4 behavior.
		callerIdx := idx - 1
		if callerIdx >= 0 {
			info.Name, info.NameWhat = vm.funcNameFromCall(&stack[callerIdx])
		}

		// If bytecode-based name inference failed, use the frame's override name
		// (e.g., "close" for __close metamethod calls).
		// Suppress the name for error-triggered __close (suppressTracebackName),
		// matching Lua 5.4 where error-path closes report nil name.
		if info.NameWhat == "" && frame.callName != "" && !frame.suppressTracebackName {
			info.Name = frame.callName
			info.NameWhat = frame.callNameWhat
		}
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
		info.NUps = fn.NativeFuncNups()
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

		info.ActiveLines = activeLines(proto)

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

		// If a forward jump can bypass this instruction and still reach the
		// CALL, the name is path-dependent (e.g., from or/and expressions).
		// Suppress annotation in that case, matching Lua 5.4's filterpc.
		if hasForwardBypass(proto, i, pc) {
			return "", ""
		}

		switch prevOp {
		case compiler.OP_GETTABUP:
			// R[A] := UpValue[B][K[C]:string]
			c := prev.C()
			if c < len(proto.Constants) && proto.Constants[c].Type == compiler.ValString {
				if isUpvalIdx_ENV(proto, prev.B()) {
					return proto.Constants[c].SVal, "global"
				}
				return proto.Constants[c].SVal, "field"
			}
			return "", ""
		case compiler.OP_GETI:
			return "integer index", "field"
		case compiler.OP_GETTABLE:
			// R[A] = R[B][R[C]] — resolve key name from register C.
			b := prev.B()
			c := prev.C()
			kn := kName(proto, i, c)
			// Check if table is _ENV (global access with high constant index).
			if localName(proto, b, i) == "_ENV" || isUpvalEnv(proto, i, b) {
				if kn != "" {
					return kn, "global"
				}
				return "?", "global"
			}
			// Detect SELF fallback pattern: MOVE base+1,obj + LOADK + GETTABLE base,obj,key.
			// If the previous instruction wrote obj to base+1 (self), this is a method call.
			if kn != "" && isSelfFallback(proto, i, reg, b) {
				return kn, "method"
			}
			if kn != "" {
				return kn, "field"
			}
			return "?", "field"
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
				tag := decodeBytecodeMetamethodTag(next.C())
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
				tag := decodeBytecodeMetamethodTag(next.C())
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
				tag := decodeBytecodeMetamethodTag(next.C())
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
				tag := decodeBytecodeMetamethodTag(next.C())
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
	stack, active := vm.debugCallStack()
	idx := len(stack) - 1 - level
	if idx < 0 || idx >= len(stack) {
		return "", Nil, false
	}

	frame := &stack[idx]
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
		// Upper bound: next frame's base (if any) or vm.top.
		// C Lua uses ci->top which extends to func+LUA_MINSTACK+1,
		// so C frames always expose slots beyond argc.  For the top
		// frame (e.g. yield in a suspended coroutine) use vm.top.
		limit := frame.base + 1 + frame.argc
		if vm.inHook {
			limit += frame.ntransfer
		}
		if idx+1 < len(stack) {
			if stack[idx+1].base < limit {
				limit = stack[idx+1].base
			}
		} else if vm.top > limit {
			limit = vm.top
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
		val := Nil
		if frame.varargs != nil && varIdx < len(frame.varargs) {
			val = frame.varargs[varIdx]
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

	stackIdx := frame.base + reg
	limit := vm.frameStackLimit(stack, idx, proto.MaxStack, active)
	if stackIdx < frame.base || stackIdx >= limit {
		return "", Nil, false
	}
	if name == "?" {
		// Lua 5.4 exposes unnamed registers within the frame as "(temporary)"
		// for both the current frame and non-current frames, not just hooks.
		// The limit check above already ensures we're within the valid range.
		name = "(temporary)"
	}
	val := Nil
	if stackIdx >= 0 && stackIdx < len(vm.stack) {
		val = vm.stack[stackIdx]
	}

	return name, val, true
}

func (vm *VM) frameStackLimit(stack []callFrame, idx, maxStack int, active bool) int {
	frame := &stack[idx]

	// Current frame (topmost): limit = vm.top, matching C Lua's L->top.p.
	if idx+1 >= len(stack) {
		if active {
			limit := vm.top
			// Ensure at least maxStack registers are accessible
			frameLimit := frame.base + maxStack
			if frameLimit > limit {
				limit = frameLimit
			}
			return limit
		}
		return frame.base + maxStack
	}

	// Non-current frame: match C Lua's ci->next->func.p.
	// For Lua next frames, base = caller.base + MaxStack (frames don't
	// overlap in GoLua), which is too large. For native next frames
	// reached via OP_CALL, stack[idx+1].base equals the CALL register
	// and matches; but for TAILCALL->native the native frame is placed
	// at vm.top (above the caller's call-register temp), so
	// nextFrame.base would expose the caller's call-temp as a local.
	// In both cases the right boundary is the call register, which is
	// the A field of the CALL/TAILCALL instruction in the caller's
	// bytecode. Derive it from there whenever the caller is a Lua frame.
	nextFrame := &stack[idx+1]
	if frame.closure != nil {
		limit := frame.base + maxStack
		proto := frame.closure.Proto
		pc := frame.pc - 1
		if pc >= 0 && pc < len(proto.Code) {
			inst := proto.Code[pc]
			op := inst.OpCode()
			if op == compiler.OP_CALL || op == compiler.OP_TAILCALL {
				limit = frame.base + inst.A()
			}
		}
		// During return hooks, the transfer info indicates how many
		// return values are visible on the stack. Extend the limit
		// to cover them, matching Lua 5.4's L->top during return.
		if frame.ftransfer > 0 {
			transferEnd := frame.base + frame.ftransfer - 1 + frame.ntransfer
			if transferEnd > limit {
				limit = transferEnd
			}
		}
		return limit
	}

	return nextFrame.base
}

func activeLines(proto *compiler.Proto) map[int]bool {
	lines := make(map[int]bool)
	// Lua 5.4 simply reports all lines that have at least one instruction.
	// No filtering by opcode type — every instruction's line is active.
	for pc, line := range proto.Lines {
		if line <= 0 || pc >= len(proto.Code) {
			continue
		}
		lines[line] = true
	}
	if proto.LastLine > 0 && len(proto.Lines) > 0 {
		lines[proto.LastLine] = true
	}
	return lines
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
		return "", false
	}
	return name, true
}

// SetLocal sets the value of local variable #index at the given stack level.
// Returns the name of the variable, or ("", false) if out of range.
func (vm *VM) SetLocal(level, index int, val Value) (string, bool) {
	stack, active := vm.debugCallStack()
	idx := len(stack) - 1 - level
	if idx < 0 || idx >= len(stack) {
		return "", false
	}

	frame := &stack[idx]
	if frame.closure == nil {
		if index <= 0 {
			return "", false
		}
		stackIdx := frame.base + index
		if stackIdx < 0 || stackIdx >= len(vm.stack) {
			return "", false
		}
		// Determine the upper bound for accessible slots.
		// During hooks on the active VM, the native frame's args + transfer
		// area are accessible. For suspended coroutines, Lua 5.4's db_setlocal
		// uses the caller's L->top which in practice allows exactly 1 slot
		// (index 1 is always reachable for C frames).
		argc := frame.argc
		if argc < 1 {
			argc = 1 // at least 1 slot accessible (Lua 5.4 uses caller's L->top)
		}
		limit := frame.base + 1 + argc
		if active && vm.inHook {
			limit += frame.ntransfer
		}
		if idx+1 < len(stack) {
			nextBase := stack[idx+1].base
			if nextBase > limit {
				limit = nextBase
			}
		} else if active && vm.top > limit {
			limit = vm.top
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
		if frame.varargs != nil && varIdx < len(frame.varargs) {
			frame.varargs[varIdx] = val
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

	stackIdx := frame.base + reg
	limit := vm.frameStackLimit(stack, idx, proto.MaxStack, active)
	if stackIdx < frame.base || stackIdx >= limit {
		return "", false
	}
	if name == "?" {
		// Lua 5.4 exposes unnamed registers within the frame as "(temporary)"
		// for both the current frame and non-current frames, not just hooks.
		name = "(temporary)"
	}
	if stackIdx >= 0 && stackIdx < len(vm.stack) {
		vm.stack[stackIdx] = val
	}

	return name, true
}

// IsValidLevel checks if a stack level is within the current call stack bounds.
func (vm *VM) IsValidLevel(level int) bool {
	stack, _ := vm.debugCallStack()
	idx := len(stack) - 1 - level
	return idx >= 0 && idx < len(stack)
}

// GetRegistry returns the VM's registry table, creating it on first access.
// Per Lua 5.4, registry[1] is the main thread and registry[2] is the global table.
func (vm *VM) GetRegistry() LuaTable {
	if vm.registry == nil {
		vm.registry = NewEmptyTable()

		// Create _HOOKKEY: a table with weak keys metatable (__mode = 'k').
		// In C Lua this maps threads to their hook functions; GoLua stores
		// hooks differently, but the entry must exist for conformance.
		hookKey := NewEmptyTable()
		hookMt := NewEmptyTable()
		hookMt.SetString("__mode", NewString("k"))
		hookKey.SetMetatable(hookMt)
		_ = vm.registry.Set(NewString("_HOOKKEY"), NewTable(hookKey))
	}
	// Ensure standard entries are populated when available.
	if !vm.threadObj.IsNil() {
		_ = vm.registry.Set(NewInt(1), vm.threadObj)
	}
	if vm.globals != nil {
		_ = vm.registry.Set(NewInt(2), NewTable(vm.globals))
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
		// If a forward jump can bypass this instruction and still reach the
		// use site, the assignment is path-dependent (e.g., or/and expressions).
		// Suppress annotation, matching Lua 5.4's filterpc in findsetreg.
		if hasForwardBypass(proto, i, pc) {
			return "", ""
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
				if isUpvalIdx_ENV(proto, inst.B()) {
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
			// R[A] = R[B][R[C]] — resolve key name from register C.
			b := inst.B()
			c := inst.C()
			kn := kName(proto, i, c)
			// Check if table is _ENV (global access with high constant index).
			if localName(proto, b, i) == "_ENV" || isUpvalEnv(proto, i, b) {
				if kn != "" {
					return kn, "global"
				}
				return "?", "global"
			}
			// Detect SELF fallback pattern (MOVE base+1,obj + LOADK + GETTABLE).
			if kn != "" && isSelfFallback(proto, i, reg, b) {
				return kn, "method"
			}
			if kn != "" {
				return kn, "field"
			}
			return "?", "field"
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

// hasForwardBypass reports whether a forward jump before defPC can skip the
// assignment at defPC and still reach usePC. In that case, the assignment is
// path-dependent and should not be treated as the definite source for naming.
func hasForwardBypass(proto *compiler.Proto, defPC, usePC int) bool {
	if defPC <= 0 || usePC <= defPC {
		return false
	}
	for j := defPC - 1; j >= 0; j-- {
		inst := proto.Code[j]
		if inst.OpCode() != compiler.OP_JMP {
			continue
		}
		target := j + 1 + inst.SJ()
		if target > defPC && target <= usePC {
			return true
		}
	}
	return false
}

// isInternalName returns true if the name is a compiler-generated internal
// name (starts with '(' such as "(for state)" or "(for control)").
func isInternalName(name string) bool {
	return len(name) > 0 && name[0] == '('
}

// isUpvalIdx_ENV checks whether the given upvalue index refers to _ENV
// by checking the upvalue's name in the prototype.
func isUpvalIdx_ENV(proto *compiler.Proto, idx int) bool {
	return idx < len(proto.Upvalues) && proto.Upvalues[idx].Name == "_ENV"
}

// isUpvalEnv checks whether the register at the given PC was loaded via
// GETUPVAL from _ENV. Used to distinguish "global" from "field"
// when GETTABLE is the fallback for GETTABUP with large constant indices.
func isUpvalEnv(proto *compiler.Proto, pc int, reg int) bool {
	for i := pc - 1; i >= 0; i-- {
		inst := proto.Code[i]
		if inst.A() != reg {
			continue
		}
		op := inst.OpCode()
		if op == compiler.OP_GETUPVAL && isUpvalIdx_ENV(proto, inst.B()) {
			return true
		}
		// SETFIELD/SETTABLE/SETI use R[A] as the table being written to,
		// not as a destination register. Skip these so we can keep scanning
		// backward to find the GETUPVAL that loaded _ENV into this register.
		if op == compiler.OP_SETFIELD || op == compiler.OP_SETTABLE || op == compiler.OP_SETI {
			continue
		}
		// Also check if the register is a local named _ENV.
		if localName(proto, reg, i) == "_ENV" {
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
			if hasForwardBypass(proto, i, pc) {
				continue
			}
			bx := inst.Bx()
			if bx < len(proto.Constants) && proto.Constants[bx].Type == compiler.ValString {
				return proto.Constants[bx].SVal
			}
			return ""
		case compiler.OP_LOADKX:
			if hasForwardBypass(proto, i, pc) {
				continue
			}
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

// isSelfFallback detects the SELF fallback pattern emitted by the compiler
// when a method name's constant index exceeds MaxArgC. The pattern is:
//
//	MOVE base+1, objReg    // copy self
//	LOADK/LOADKX tmp, "method_name"
//	GETTABLE base, objReg, tmp
//
// Returns true if the instruction at gettablePC is part of this pattern.
func isSelfFallback(proto *compiler.Proto, gettablePC int, base int, objReg int) bool {
	// Look backward past LOADK/LOADKX to find a MOVE writing to base+1
	for j := gettablePC - 1; j >= 0; j-- {
		jInst := proto.Code[j]
		jOp := jInst.OpCode()
		switch jOp {
		case compiler.OP_LOADK, compiler.OP_LOADKX:
			// Skip LOADK that loaded the key into the temp register
			continue
		case compiler.OP_EXTRAARG:
			// Part of LOADKX
			continue
		case compiler.OP_MOVE:
			// Check: MOVE base+1, objReg
			return jInst.A() == base+1 && jInst.B() == objReg
		default:
			return false
		}
	}
	return false
}
