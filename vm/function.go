package vm

import (
	"github.com/iceisfun/golua/compiler"
)

// closureInlineUpvalues sizes the inline upvalue slot count. Closures with
// at most this many upvalues skip the make([]*Upvalue) heap allocation by
// pointing Upvalues at the inline buffer. Most closures in real-world Lua
// code capture 0–2 upvalues, so 4 covers the long tail without inflating
// every Closure too much.
const closureInlineUpvalues = 4

// Closure represents a Lua closure: a function prototype paired with its
// captured upvalues. Each closure instance shares the same [compiler.Proto]
// but has its own upvalue bindings, allowing closures created at different
// call sites to capture different variables.
//
// Lua 5.4 Reference: §3.4.11 (function definitions), §2.5.1 (upvalues).
type Closure struct {
	Proto       *compiler.Proto // compiled bytecode and metadata
	Upvalues    []*Upvalue      // captured variables from enclosing scopes
	constValues []Value         // cached conversion of Proto.Constants to vm.Value

	// inlineUpvalues backs Upvalues for the common small-upvalue-count case.
	// When len(proto.Upvalues) <= closureInlineUpvalues, NewClosure points
	// Upvalues at this array instead of allocating a separate slice.
	inlineUpvalues [closureInlineUpvalues]*Upvalue
}

// NewClosure creates a new closure from a prototype.
func NewClosure(proto *compiler.Proto) *Closure {
	cl := &Closure{Proto: proto}
	nups := len(proto.Upvalues)
	if nups <= closureInlineUpvalues {
		cl.Upvalues = cl.inlineUpvalues[:nups]
	} else {
		cl.Upvalues = make([]*Upvalue, nups)
	}
	return cl
}

// ConstValues returns the cached runtime Value conversions of the Proto's constants.
// The cache is built on first call and reused thereafter, avoiding repeated
// NewString allocations when loading string constants in hot loops.
func (cl *Closure) ConstValues() []Value {
	if cl.constValues == nil {
		consts := cl.Proto.Constants
		cl.constValues = make([]Value, len(consts))
		for i, c := range consts {
			switch c.Type {
			case compiler.ValNil:
				cl.constValues[i] = Nil
			case compiler.ValFalse:
				cl.constValues[i] = False
			case compiler.ValTrue:
				cl.constValues[i] = True
			case compiler.ValInt:
				cl.constValues[i] = NewInt(c.IVal)
			case compiler.ValFloat:
				cl.constValues[i] = NewFloat(c.FVal)
			case compiler.ValString:
				cl.constValues[i] = NewString(c.SVal)
			}
		}
	}
	return cl.constValues
}

// NativeFunc is a Go function callable from Lua.
// It receives the VM state and returns the number of return values.
// Arguments are on the stack starting at vm.Base().
// Return values should be placed starting at vm.Base().
type NativeFunc func(vm *VM) int

// Upvalue represents a captured variable from an enclosing scope.
// Open upvalues reference a live stack slot (shared with the enclosing
// function). When the enclosing scope exits, the upvalue is "closed" —
// the value is copied from the stack into the Upvalue itself, and the
// stack reference is released.
//
// Lua 5.4 Reference: §3.5 (visibility rules).
type Upvalue struct {
	vm       *VM   // Reference to VM for open upvalues
	stackIdx int   // Absolute stack index for open upvalues
	closed   Value // Holds the value when closed
	isOpen   bool  // true while referencing a live stack slot
}

// NewOpenUpvalue creates an open upvalue pointing to a stack slot.
func NewOpenUpvalue(vm *VM, idx int) *Upvalue {
	return &Upvalue{
		vm:       vm,
		stackIdx: idx,
		isOpen:   true,
	}
}

// Get returns the upvalue's current value.
func (u *Upvalue) Get() Value {
	if u.isOpen {
		return u.vm.stack[u.stackIdx]
	}
	return u.closed
}

// Set sets the upvalue's value.
func (u *Upvalue) Set(v Value) {
	if u.isOpen {
		u.vm.stack[u.stackIdx] = v
	} else {
		u.closed = v
	}
}

// Close closes the upvalue, copying the value from the stack.
func (u *Upvalue) Close() {
	if u.isOpen {
		u.closed = u.vm.stack[u.stackIdx]
		u.vm = nil // Release VM reference
		u.isOpen = false
	}
}

// IsOpen returns true if the upvalue is still open (pointing to stack).
func (u *Upvalue) IsOpen() bool {
	return u.isOpen
}

// StackIndex returns the stack index for an open upvalue.
func (u *Upvalue) StackIndex() int {
	return u.stackIdx
}

// SetClosed creates a closed upvalue with the given value.
func (u *Upvalue) SetClosed(v Value) {
	u.closed = v
	u.isOpen = false
	u.vm = nil
}
