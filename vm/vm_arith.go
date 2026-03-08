package vm

import (
	"math"

	"github.com/iceisfun/golua/compiler"
)

// luaNumMod computes Lua's float modulo: result = fmod(a, b), then adjusts
// the sign to match b when they differ. When the result is NaN, it returns
// negative NaN to match C's fmod behavior (Go's math.Mod returns positive NaN).
func luaNumMod(a, b float64) float64 {
	result := math.Mod(a, b)
	if result != 0 && (result < 0) != (b < 0) {
		result += b
	}
	if math.IsNaN(result) {
		return math.Copysign(math.NaN(), -1)
	}
	return result
}

// arith performs a register-register arithmetic operation.
// Lua 5.4.6+: strings are NOT coerced at the VM level; coercion is handled
// by the string metatable's arithmetic metamethods.
func (vm *VM) arith(op compiler.OpCode, v1, v2 Value, regB, regC int) (Value, error) {
	// Float fast path: both operands are already floats (common in numeric code)
	if v1.typ == typeFloat && v2.typ == typeFloat {
		n1, n2 := v1.num, v2.num
		switch op {
		case compiler.OP_ADD:
			return NewFloat(n1 + n2), nil
		case compiler.OP_SUB:
			return NewFloat(n1 - n2), nil
		case compiler.OP_MUL:
			return NewFloat(n1 * n2), nil
		case compiler.OP_DIV:
			return NewFloat(n1 / n2), nil
		case compiler.OP_IDIV:
			return NewFloat(math.Floor(n1 / n2)), nil
		case compiler.OP_MOD:
			return NewFloat(luaNumMod(n1, n2)), nil
		case compiler.OP_POW:
			return NewFloat(math.Pow(n1, n2)), nil
		}
	}

	// Integer fast path: both operands are int (not strings)
	if v1.IsInt() && v2.IsInt() && op != compiler.OP_DIV && op != compiler.OP_POW {
		i1, i2 := v1.AsInt(), v2.AsInt()
		switch op {
		case compiler.OP_ADD:
			return NewInt(i1 + i2), nil
		case compiler.OP_SUB:
			return NewInt(i1 - i2), nil
		case compiler.OP_MUL:
			return NewInt(i1 * i2), nil
		case compiler.OP_IDIV:
			if i2 == 0 {
				return Nil, vm.runtimeError("attempt to divide by zero")
			}
			if i2 == -1 {
				return NewInt(-i1), nil
			}
			q := i1 / i2
			// Lua floor division: correct toward negative infinity
			if (i1^i2) < 0 && q*i2 != i1 {
				q--
			}
			return NewInt(q), nil
		case compiler.OP_MOD:
			if i2 == 0 {
				return Nil, vm.runtimeError("attempt to perform 'n%%0'")
			}
			if i2 == -1 {
				return NewInt(0), nil
			}
			r := i1 % i2
			if r != 0 && (r^i2) < 0 {
				r += i2
			}
			return NewInt(r), nil
		}
	}

	// Mixed number types (int + float or float + int): promote to float
	if v1.IsNumber() && v2.IsNumber() {
		n1, _ := v1.ToNumber()
		n2, _ := v2.ToNumber()
		var result float64
		switch op {
		case compiler.OP_ADD:
			result = n1 + n2
		case compiler.OP_SUB:
			result = n1 - n2
		case compiler.OP_MUL:
			result = n1 * n2
		case compiler.OP_DIV:
			result = n1 / n2
		case compiler.OP_IDIV:
			result = math.Floor(n1 / n2)
		case compiler.OP_MOD:
			result = luaNumMod(n1, n2)
		case compiler.OP_POW:
			result = math.Pow(n1, n2)
		}
		return NewFloat(result), nil
	}

	// Try metamethods (handles strings via string metatable, tables via their metatable)
	mmName := vm.arithMetamethod(op)
	if mm := vm.getArithMetamethod(v1, v2, mmName); !mm.IsNil() {
		if !mm.IsCallable() {
			return Nil, vm.runtimeError("attempt to call a %s value (metamethod '%s')", mm.Type(), mmName[2:])
		}
		result, err := vm.callMetamethod(mmName[2:], mm, v1, v2)
		if err != nil {
			return Nil, err
		}
		return result, nil
	}

	// No metamethod found, report error
	if !v1.IsNumber() {
		return Nil, vm.runtimeError("attempt to perform arithmetic on a %s value%s", vm.ObjTypeName(v1), vm.varInfo(regB))
	}
	return Nil, vm.runtimeError("attempt to perform arithmetic on a %s value%s", vm.ObjTypeName(v2), vm.varInfo(regC))
}

// arithK performs a register-constant arithmetic operation.
// Lua 5.4.6+: strings are NOT coerced at the VM level.
func (vm *VM) arithK(op compiler.OpCode, v, kv Value, regB int) (Value, error) {
	// Float fast path: both operands are already floats
	if v.typ == typeFloat && kv.typ == typeFloat {
		n1, n2 := v.num, kv.num
		switch op {
		case compiler.OP_ADDK:
			return NewFloat(n1 + n2), nil
		case compiler.OP_SUBK:
			return NewFloat(n1 - n2), nil
		case compiler.OP_MULK:
			return NewFloat(n1 * n2), nil
		case compiler.OP_DIVK:
			return NewFloat(n1 / n2), nil
		case compiler.OP_IDIVK:
			return NewFloat(math.Floor(n1 / n2)), nil
		case compiler.OP_MODK:
			return NewFloat(luaNumMod(n1, n2)), nil
		case compiler.OP_POWK:
			return NewFloat(math.Pow(n1, n2)), nil
		}
	}

	// Integer fast path
	if v.IsInt() && kv.IsInt() && op != compiler.OP_DIVK && op != compiler.OP_POWK {
		i1, i2 := v.AsInt(), kv.AsInt()
		switch op {
		case compiler.OP_ADDK:
			return NewInt(i1 + i2), nil
		case compiler.OP_SUBK:
			return NewInt(i1 - i2), nil
		case compiler.OP_MULK:
			return NewInt(i1 * i2), nil
		case compiler.OP_IDIVK:
			if i2 == 0 {
				return Nil, vm.runtimeError("attempt to divide by zero")
			}
			if i2 == -1 {
				return NewInt(-i1), nil
			}
			q := i1 / i2
			if (i1^i2) < 0 && q*i2 != i1 {
				q--
			}
			return NewInt(q), nil
		case compiler.OP_MODK:
			if i2 == 0 {
				return Nil, vm.runtimeError("attempt to perform 'n%%0'")
			}
			if i2 == -1 {
				return NewInt(0), nil
			}
			r := i1 % i2
			if r != 0 && (r^i2) < 0 {
				r += i2
			}
			return NewInt(r), nil
		}
	}

	// Mixed number types (int + float or float + int): promote to float
	if v.IsNumber() && kv.IsNumber() {
		n1, _ := v.ToNumber()
		n2, _ := kv.ToNumber()
		var result float64
		switch op {
		case compiler.OP_ADDK:
			result = n1 + n2
		case compiler.OP_SUBK:
			result = n1 - n2
		case compiler.OP_MULK:
			result = n1 * n2
		case compiler.OP_DIVK:
			result = n1 / n2
		case compiler.OP_IDIVK:
			result = math.Floor(n1 / n2)
		case compiler.OP_MODK:
			result = luaNumMod(n1, n2)
		case compiler.OP_POWK:
			result = math.Pow(n1, n2)
		}
		return NewFloat(result), nil
	}

	// Try metamethods
	mmName := vm.arithMetamethod(op)
	if mm := vm.getArithMetamethod(v, kv, mmName); !mm.IsNil() {
		if !mm.IsCallable() {
			return Nil, vm.runtimeError("attempt to call a %s value (metamethod '%s')", mm.Type(), mmName[2:])
		}
		result, err := vm.callMetamethod(mmName[2:], mm, v, kv)
		if err != nil {
			return Nil, err
		}
		return result, nil
	}

	if !v.IsNumber() {
		return Nil, vm.runtimeError("attempt to perform arithmetic on a %s value%s", vm.ObjTypeName(v), vm.varInfo(regB))
	}
	return Nil, vm.runtimeError("attempt to perform arithmetic on a %s value", vm.ObjTypeName(kv))
}

// arithMetamethod returns the metamethod name for an arithmetic opcode
func (vm *VM) arithMetamethod(op compiler.OpCode) string {
	switch op {
	case compiler.OP_ADD, compiler.OP_ADDK:
		return "__add"
	case compiler.OP_SUB, compiler.OP_SUBK:
		return "__sub"
	case compiler.OP_MUL, compiler.OP_MULK:
		return "__mul"
	case compiler.OP_DIV, compiler.OP_DIVK:
		return "__div"
	case compiler.OP_IDIV, compiler.OP_IDIVK:
		return "__idiv"
	case compiler.OP_MOD, compiler.OP_MODK:
		return "__mod"
	case compiler.OP_POW, compiler.OP_POWK:
		return "__pow"
	default:
		return ""
	}
}

// bitwise performs a register-register bitwise operation, with metamethod fallback.
func (vm *VM) bitwise(op compiler.OpCode, v1, v2 Value, regB, regC int) (Value, error) {
	// Lua 5.4: bitwise ops do NOT coerce strings (unlike arithmetic)
	var i1, i2 int64
	var ok1, ok2 bool
	if !v1.IsString() {
		i1, ok1 = v1.ToInt()
	}
	if !v2.IsString() {
		i2, ok2 = v2.ToInt()
	}
	if ok1 && ok2 {
		var result int64
		switch op {
		case compiler.OP_BAND:
			result = i1 & i2
		case compiler.OP_BOR:
			result = i1 | i2
		case compiler.OP_BXOR:
			result = i1 ^ i2
		case compiler.OP_SHL:
			if i2 >= 0 {
				result = i1 << uint(i2)
			} else {
				result = int64(uint64(i1) >> uint(-i2))
			}
		case compiler.OP_SHR:
			if i2 >= 0 {
				result = int64(uint64(i1) >> uint(i2))
			} else {
				result = i1 << uint(-i2)
			}
		}
		return NewInt(result), nil
	}

	// Try metamethods
	mmName := vm.bitwiseMetamethod(op)
	if mm := vm.getArithMetamethod(v1, v2, mmName); !mm.IsNil() {
		if !mm.IsCallable() {
			return Nil, vm.runtimeError("attempt to call a %s value (metamethod '%s')", mm.Type(), mmName[2:])
		}
		return vm.callMetamethod(mmName[2:], mm, v1, v2)
	}

	if !ok1 {
		if v1.IsNumber() {
			return Nil, vm.runtimeErrorForNumber(regB)
		}
		return Nil, vm.runtimeError("attempt to perform bitwise operation on a %s value%s", vm.ObjTypeName(v1), vm.varInfo(regB))
	}
	if v2.IsNumber() {
		return Nil, vm.runtimeErrorForNumber(regC)
	}
	return Nil, vm.runtimeError("attempt to perform bitwise operation on a %s value%s", vm.ObjTypeName(v2), vm.varInfo(regC))
}

// bitwiseK performs a register-constant bitwise operation.
func (vm *VM) bitwiseK(op compiler.OpCode, v, kv Value, regB int) (Value, error) {
	// Lua 5.4: bitwise ops do NOT coerce strings
	var i1, i2 int64
	var ok1, ok2 bool
	if !v.IsString() {
		i1, ok1 = v.ToInt()
	}
	if !kv.IsString() {
		i2, ok2 = kv.ToInt()
	}
	if ok1 && ok2 {
		var result int64
		switch op {
		case compiler.OP_BANDK:
			result = i1 & i2
		case compiler.OP_BORK:
			result = i1 | i2
		case compiler.OP_BXORK:
			result = i1 ^ i2
		}
		return NewInt(result), nil
	}

	// Try metamethods
	mmName := vm.bitwiseMetamethod(op)
	if mm := vm.getArithMetamethod(v, kv, mmName); !mm.IsNil() {
		if !mm.IsCallable() {
			return Nil, vm.runtimeError("attempt to call a %s value (metamethod '%s')", mm.Type(), mmName[2:])
		}
		return vm.callMetamethod(mmName[2:], mm, v, kv)
	}

	if !ok1 {
		if v.IsNumber() {
			return Nil, vm.runtimeErrorForNumber(regB)
		}
		return Nil, vm.runtimeError("attempt to perform bitwise operation on a %s value%s", vm.ObjTypeName(v), vm.varInfo(regB))
	}
	if kv.IsNumber() {
		return Nil, vm.runtimeError("number has no integer representation")
	}
	return Nil, vm.runtimeError("attempt to perform bitwise operation on a %s value", vm.ObjTypeName(kv))
}

// bitwiseMetamethod returns the metamethod name for a bitwise opcode
func (vm *VM) bitwiseMetamethod(op compiler.OpCode) string {
	switch op {
	case compiler.OP_BAND, compiler.OP_BANDK:
		return "__band"
	case compiler.OP_BOR, compiler.OP_BORK:
		return "__bor"
	case compiler.OP_BXOR, compiler.OP_BXORK:
		return "__bxor"
	case compiler.OP_SHL:
		return "__shl"
	case compiler.OP_SHR:
		return "__shr"
	default:
		return ""
	}
}

// getArithMetamethod looks for an arithmetic metamethod on either operand
func (vm *VM) getArithMetamethod(v1, v2 Value, name string) Value {
	// Try first operand
	if mm := vm.getMetafield(v1, name); !mm.IsNil() {
		return mm
	}
	// Try second operand
	if mm := vm.getMetafield(v2, name); !mm.IsNil() {
		return mm
	}
	return Nil
}
