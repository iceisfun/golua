package vm

import (
	"github.com/iceisfun/golua/compiler"
)

// Closure represents a Lua closure (function + captured upvalues).
type Closure struct {
	Proto    *compiler.Proto
	Upvalues []*Upvalue
}

// NewClosure creates a new closure from a prototype.
func NewClosure(proto *compiler.Proto) *Closure {
	return &Closure{
		Proto:    proto,
		Upvalues: make([]*Upvalue, len(proto.Upvalues)),
	}
}

// NativeFunc is a Go function callable from Lua.
// It receives the VM state and returns the number of return values.
// Arguments are on the stack starting at vm.Base().
// Return values should be placed starting at vm.Base().
type NativeFunc func(vm *VM) int

// Upvalue represents a captured variable.
// When open, it references a stack slot via the VM.
// When closed, it holds the value directly.
type Upvalue struct {
	vm       *VM   // Reference to VM for open upvalues
	stackIdx int   // Absolute stack index for open upvalues
	closed   Value // Holds the value when closed
	isOpen   bool
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
