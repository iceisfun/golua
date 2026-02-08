package vm

import (
	"fmt"
	"strings"
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
			name := vm.frameFuncName(frame)
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
