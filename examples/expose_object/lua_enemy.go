// lua_enemy.go: Go → Lua adapter.
//
// Converts an *Enemy into a Lua table of functions.
// Each method is explicit — no reflection, no automatic binding.
// Internal fields are never exposed.
package main

import "github.com/iceisfun/golua/v1/vm"

// EnemyToLua converts a Go Enemy into a Lua table of closures.
// The returned table has read-only accessors and explicit mutation methods.
// Each closure captures the *Enemy pointer, so mutations are visible
// immediately from both Go and Lua.
func EnemyToLua(e *Enemy) *vm.Table {
	t := vm.NewEmptyTable()

	// --- Accessors ---

	t.SetString("name", vm.NewNativeFunc(func(v *vm.VM) int {
		v.Set(0, vm.NewString(e.Name()))
		return 1
	}))

	t.SetString("health", vm.NewNativeFunc(func(v *vm.VM) int {
		v.Set(0, vm.NewInt(int64(e.Health())))
		return 1
	}))

	t.SetString("max_hp", vm.NewNativeFunc(func(v *vm.VM) int {
		v.Set(0, vm.NewInt(int64(e.MaxHP())))
		return 1
	}))

	t.SetString("position", vm.NewNativeFunc(func(v *vm.VM) int {
		x, y := e.Position()
		v.Set(0, vm.NewFloat(x))
		v.Set(1, vm.NewFloat(y))
		return 2
	}))

	t.SetString("is_alive", vm.NewNativeFunc(func(v *vm.VM) int {
		if e.IsAlive() {
			v.Set(0, vm.True)
		} else {
			v.Set(0, vm.False)
		}
		return 1
	}))

	// --- Mutators ---

	t.SetString("take_damage", vm.NewNativeFunc(func(v *vm.VM) int {
		amount := int(v.Get(1).AsInt())
		e.TakeDamage(amount)
		return 0
	}))

	t.SetString("heal", vm.NewNativeFunc(func(v *vm.VM) int {
		amount := int(v.Get(1).AsInt())
		e.Heal(amount)
		return 0
	}))

	t.SetString("move_to", vm.NewNativeFunc(func(v *vm.VM) int {
		x := v.Get(1).AsFloat()
		y := v.Get(2).AsFloat()
		e.MoveTo(x, y)
		return 0
	}))

	return t
}
