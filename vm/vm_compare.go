package vm

// equal checks for equality, handling __eq metamethod
func (vm *VM) equal(v1, v2 Value) (bool, error) {
	// 1. If types are different and not numbers (int/float), false
	if v1.typ != v2.typ && !v1.IsNumber() && !v2.IsNumber() {
		return false, nil // Standard Lua behavior: different types are unequal
	}

	// 2. Raw equality
	if v1.Equal(v2) {
		return true, nil
	}

	// 3. Userdata/Table check for __eq
	// Lua 5.4: try left operand's __eq first, then right's.
	// __eq is only called for tables and full userdata, NOT threads.
	leftHasEqMeta := (v1.IsTable() && !v1.isThread()) || v1.IsUserdata()
	rightHasEqMeta := (v2.IsTable() && !v2.isThread()) || v2.IsUserdata()
	if leftHasEqMeta && rightHasEqMeta {
		mm := vm.getMetafield(v1, "__eq")
		if mm.IsNil() {
			mm = vm.getMetafield(v2, "__eq")
		}
		if mm.IsNil() {
			return false, nil
		}

		res, err := vm.callMetamethod("eq", mm, v1, v2)
		if err != nil {
			return false, err
		}
		return res.ToBool(), nil
	}

	return false, nil
}

// CompareLT performs a less-than comparison, honoring __lt metamethods.
func (vm *VM) CompareLT(v1, v2 Value) (bool, error) {
	return vm.lessThan(v1, v2)
}

// lessThan checks for less than, handling __lt metamethod
func (vm *VM) lessThan(v1, v2 Value) (bool, error) {
	// 1. Primitive comparison
	if res, ok := v1.LessThan(v2); ok {
		return res, nil
	}

	// 2. Metamethod __lt
	op := "__lt"
	mm := vm.getMetafield(v1, op)
	if mm.IsNil() {
		mm = vm.getMetafield(v2, op)
	}

	if !mm.IsNil() {
		res, err := vm.callMetamethod("lt", mm, v1, v2)
		if err != nil {
			return false, err
		}
		return res.ToBool(), nil
	}

	return false, vm.compareError(v1, v2)
}

// lessEqual checks for less equal, handling __le metamethod
func (vm *VM) lessEqual(v1, v2 Value) (bool, error) {
	// 1. Primitive comparison
	if res, ok := v1.LessEqual(v2); ok {
		return res, nil
	}

	// 2. Metamethod __le
	op := "__le"
	mm := vm.getMetafield(v1, op)
	if mm.IsNil() {
		mm = vm.getMetafield(v2, op)
	}

	if !mm.IsNil() {
		res, err := vm.callMetamethod("le", mm, v1, v2)
		if err != nil {
			return false, err
		}
		return res.ToBool(), nil
	}

	// 3. Fallback to __lt ( b < a )
	// Lua spec: if __le is not present, try __lt(b, a)
	// a <= b  ===  not (b < a)
	op = "__lt"
	mm = vm.getMetafield(v1, op)
	if mm.IsNil() {
		mm = vm.getMetafield(v2, op)
	}

	if !mm.IsNil() {
		res, err := vm.callMetamethod("lt", mm, v2, v1) // Note swapped args: b < a
		if err != nil {
			return false, err
		}
		return !res.ToBool(), nil
	}

	return false, vm.compareError(v1, v2)
}

// compareError generates the appropriate comparison error message.
// Lua 5.4: same-type uses "two TYPE values", different-type uses "TYPE with TYPE".
func (vm *VM) compareError(v1, v2 Value) error {
	t1, t2 := vm.ObjTypeName(v1), vm.ObjTypeName(v2)
	if t1 == t2 {
		return vm.runtimeError("attempt to compare two %s values", t1)
	}
	return vm.runtimeError("attempt to compare %s with %s", t1, t2)
}

// concat handles concatenation with __concat support.
// reg1 is the register of v1 (used for error messages), or -1 if unknown.
func (vm *VM) concat(v1, v2 Value, reg1 int) (Value, error) {
	// 1. Primitives (string/number)
	if (v1.IsString() || v1.IsNumber()) && (v2.IsString() || v2.IsNumber()) {
		var s1, s2 string
		if v1.IsString() {
			s1 = v1.AsString()
		} else {
			s1 = v1.String()
		}
		if v2.IsString() {
			s2 = v2.AsString()
		} else {
			s2 = v2.String()
		}
		if len(s1) > (1<<30)-len(s2) {
			return Nil, vm.runtimeError("string length overflow")
		}
		return NewString(s1 + s2), nil
	}

	// 2. Metamethod __concat
	op := "__concat"
	mm := vm.getMetafield(v1, op)
	if mm.IsNil() {
		mm = vm.getMetafield(v2, op)
	}

	if !mm.IsNil() {
		return vm.callMetamethod("concat", mm, v1, v2)
	}

	// Report the non-concatenable operand (not the valid string/number one)
	if v1.IsString() || v1.IsNumber() {
		info := ""
		if reg1 >= 0 {
			info = vm.varInfo(reg1 + 1)
		}
		return Nil, vm.runtimeError("attempt to concatenate a %s value%s", vm.ObjTypeName(v2), info)
	}
	info := ""
	if reg1 >= 0 {
		info = vm.varInfo(reg1)
	}
	return Nil, vm.runtimeError("attempt to concatenate a %s value%s", vm.ObjTypeName(v1), info)
}
