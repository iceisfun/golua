package golua

import (
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

// VM is the GoLua virtual machine. It is an alias for [vm.VM] so embedders
// can use the root package without also importing [vm].
type VM = vm.VM

// Option configures a [VM] at construction time. It is an alias for
// [vm.VMOption].
type Option = vm.VMOption

// New returns a new [VM] with an empty global environment and no standard
// library loaded. It is equivalent to [vm.New].
func New(opts ...Option) *VM {
	return vm.New(opts...)
}

// NewWithStdlib returns a new [VM] with the full standard library opened.
// It is equivalent to calling [New] followed by [OpenStdlib].
func NewWithStdlib(opts ...Option) *VM {
	v := vm.New(opts...)
	stdlib.Open(v)
	return v
}

// OpenStdlib registers the GoLua standard library (string, table, math, io,
// os, etc.) on v. It is equivalent to [stdlib.Open].
func OpenStdlib(v *VM) {
	stdlib.Open(v)
}
