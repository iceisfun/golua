package vm

import (
	"math"

	"github.com/iceisfun/golua/v2/compiler"
)

// decodeBytecodeMetamethodTag returns the MetamethodTag stored in the C field
// of an OP_MMBIN* instruction. By this point the tag is always in GoLua's
// ordinal space: locally compiled chunks emit raw TM_* ordinals, and Lua 5.4
// reference chunks are translated at undump time (see undump.go). Previously
// this function subtracted 6 for any raw >= 6, which collided with our own
// TM_IDIV (ordinal 6) and caused idiv frames to be labelled 'add' in
// tracebacks.
func decodeBytecodeMetamethodTag(raw int) compiler.MetamethodTag {
	return compiler.MetamethodTag(raw)
}

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

// shiftLeft is Lua's shift operator (lvm.c luaV_shiftl): x shifted left by y,
// where a negative y shifts right instead and a count of 64 or more in either
// direction clears the value. Go's shift operators already yield 0 for an
// out-of-range count, but they have no notion of a negative one — uint(y)
// turns a small negative count into a huge positive one.
func shiftLeft(x, y int64) int64 {
	if y < 0 {
		if y <= -64 {
			return 0
		}
		return int64(uint64(x) >> uint(-y))
	}
	if y >= 64 {
		return 0
	}
	return x << uint(y)
}

// arith performs a register-register arithmetic operation.
// Lua 5.4.6+: strings are NOT coerced at the VM level; coercion is handled
// by the string metatable's arithmetic metamethods.
func (vm *VM) arith(op compiler.OpCode, v1, v2 Value, regB, regC int) (Value, error) {
	// Float fast path: both operands are already floats (common in numeric code)
	if v1.typ == typeFloat && v2.typ == typeFloat {
		n1, n2 := v1.fval(), v2.fval()
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
			return NewFloat(PowWithSubnormalFix(n1, n2)), nil
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
			result = PowWithSubnormalFix(n1, n2)
		}
		return NewFloat(result), nil
	}

	// Try metamethods (handles strings via string metatable, tables via their metatable)
	mmName := vm.arithMetamethod(op)
	if mm := vm.getArithMetamethod(v1, v2, mmName); !mm.IsNil() {
		result, err := vm.callMetamethod(MetaEvent(mmName), mm, v1, v2)
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
//
// flip says the source wrote the constant on the left: a commutative operator
// with a constant left operand is compiled with the operands commuted so the
// *K form can be used, and the swap has to be undone before a metamethod sees
// them (ltm.c luaT_trybinassocTM). The arithmetic itself is unaffected —
// nothing but a commutative operator is ever commuted — and so are the error
// messages, which name the operand that is not a number, and the constant
// always is one.
func (vm *VM) arithK(op compiler.OpCode, v, kv Value, regB int, flip bool) (Value, error) {
	// Float fast path: both operands are already floats
	if v.typ == typeFloat && kv.typ == typeFloat {
		n1, n2 := v.fval(), kv.fval()
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
			return NewFloat(PowWithSubnormalFix(n1, n2)), nil
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
			result = PowWithSubnormalFix(n1, n2)
		}
		return NewFloat(result), nil
	}

	// Try metamethods
	mmName := vm.arithMetamethod(op)
	p1, p2 := v, kv
	if flip {
		p1, p2 = kv, v
	}
	if mm := vm.getArithMetamethod(p1, p2, mmName); !mm.IsNil() {
		result, err := vm.callMetamethod(MetaEvent(mmName), mm, p1, p2)
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
		return MetaAdd
	case compiler.OP_SUB, compiler.OP_SUBK:
		return MetaSub
	case compiler.OP_MUL, compiler.OP_MULK:
		return MetaMul
	case compiler.OP_DIV, compiler.OP_DIVK:
		return MetaDiv
	case compiler.OP_IDIV, compiler.OP_IDIVK:
		return MetaIDiv
	case compiler.OP_MOD, compiler.OP_MODK:
		return MetaMod
	case compiler.OP_POW, compiler.OP_POWK:
		return MetaPow
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
			result = shiftLeft(i1, i2)
		case compiler.OP_SHR:
			result = shiftLeft(i1, -i2)
		}
		return NewInt(result), nil
	}

	// Try metamethods
	mmName := vm.bitwiseMetamethod(op)
	if mm := vm.getArithMetamethod(v1, v2, mmName); !mm.IsNil() {
		return vm.callMetamethod(MetaEvent(mmName), mm, v1, v2)
	}

	// Match Lua 5.4 luaT_trybinTM ordering: if both operands are numbers,
	// the failure is a non-integer-representable number; otherwise the
	// failure is the non-number operand (strings never coerce for bitwise).
	if v1.IsNumber() && v2.IsNumber() {
		if !ok1 {
			return Nil, vm.runtimeErrorForNumber(regB)
		}
		return Nil, vm.runtimeErrorForNumber(regC)
	}
	if !v1.IsNumber() {
		return Nil, vm.runtimeError("attempt to perform bitwise operation on a %s value%s", vm.ObjTypeName(v1), vm.varInfo(regB))
	}
	return Nil, vm.runtimeError("attempt to perform bitwise operation on a %s value%s", vm.ObjTypeName(v2), vm.varInfo(regC))
}

// bitwiseK performs a register-constant bitwise operation. flip means the same
// thing it does in arithK: the source wrote the constant on the left.
func (vm *VM) bitwiseK(op compiler.OpCode, v, kv Value, regB int, flip bool) (Value, error) {
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
	p1, p2 := v, kv
	if flip {
		p1, p2 = kv, v
	}
	if mm := vm.getArithMetamethod(p1, p2, mmName); !mm.IsNil() {
		return vm.callMetamethod(MetaEvent(mmName), mm, p1, p2)
	}

	// Match Lua 5.4 luaT_trybinTM ordering: if both operands are numbers,
	// the failure is a non-integer-representable number; otherwise the
	// failure is the non-number operand (strings never coerce for bitwise).
	if v.IsNumber() && kv.IsNumber() {
		if !ok1 {
			return Nil, vm.runtimeErrorForNumber(regB)
		}
		return Nil, vm.runtimeError("number has no integer representation")
	}
	if !v.IsNumber() {
		return Nil, vm.runtimeError("attempt to perform bitwise operation on a %s value%s", vm.ObjTypeName(v), vm.varInfo(regB))
	}
	return Nil, vm.runtimeError("attempt to perform bitwise operation on a %s value", vm.ObjTypeName(kv))
}

// bitwiseMetamethod returns the metamethod name for a bitwise opcode
func (vm *VM) bitwiseMetamethod(op compiler.OpCode) string {
	switch op {
	case compiler.OP_BAND, compiler.OP_BANDK:
		return MetaBAnd
	case compiler.OP_BOR, compiler.OP_BORK:
		return MetaBOr
	case compiler.OP_BXOR, compiler.OP_BXORK:
		return MetaBXor
	case compiler.OP_SHL:
		return MetaShl
	case compiler.OP_SHR:
		return MetaShr
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
