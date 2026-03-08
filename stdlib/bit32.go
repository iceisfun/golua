package stdlib

import (
	"github.com/iceisfun/golua/vm"
)

func openBit32(v *vm.VM) {
	b := vm.NewEmptyTable()

	b.SetString("arshift", vm.NewNativeFunc(bit32Arshift))
	b.SetString("band", vm.NewNativeFunc(bit32Band))
	b.SetString("bnot", vm.NewNativeFunc(bit32Bnot))
	b.SetString("bor", vm.NewNativeFunc(bit32Bor))
	b.SetString("btest", vm.NewNativeFunc(bit32Btest))
	b.SetString("bxor", vm.NewNativeFunc(bit32Bxor))
	b.SetString("extract", vm.NewNativeFunc(bit32Extract))
	b.SetString("replace", vm.NewNativeFunc(bit32Replace))
	b.SetString("lrotate", vm.NewNativeFunc(bit32Lrotate))
	b.SetString("rrotate", vm.NewNativeFunc(bit32Rrotate))
	b.SetString("lshift", vm.NewNativeFunc(bit32Lshift))
	b.SetString("rshift", vm.NewNativeFunc(bit32Rshift))

	v.SetGlobal("bit32", vm.NewTable(b))
}

// toUint32 converts a Lua value to a uint32, truncating as needed.
func toUint32(v *vm.VM, idx int, fname string) uint32 {
	i := getInt(v, idx, fname)
	return uint32(i)
}

// bit32.arshift(x, disp) - arithmetic right shift
// Negative disp means left shift. Shifts >= 32 return 0 or fill with sign.
func bit32Arshift(v *vm.VM) int {
	x := toUint32(v, 1, "bit32.arshift")
	disp := getInt(v, 2, "bit32.arshift")

	var result uint32
	if disp >= 0 {
		if disp >= 32 {
			// Arithmetic: fill with sign bit
			if int32(x) < 0 {
				result = 0xFFFFFFFF
			} else {
				result = 0
			}
		} else {
			// Arithmetic shift right: cast to int32 for sign extension, then back
			result = uint32(int32(x) >> uint(disp))
		}
	} else {
		// Negative displacement = left shift
		disp = -disp
		if disp >= 32 {
			result = 0
		} else {
			result = x << uint(disp)
		}
	}

	v.Set(0, vm.NewInt(int64(result)))
	return 1
}

// bit32.band(...) - bitwise AND of all arguments. No args returns 0xFFFFFFFF.
func bit32Band(v *vm.VM) int {
	n := v.ArgCount()
	var result uint32 = 0xFFFFFFFF
	for i := 1; i <= n; i++ {
		result &= toUint32(v, i, "bit32.band")
	}
	v.Set(0, vm.NewInt(int64(result)))
	return 1
}

// bit32.bnot(x) - bitwise NOT
func bit32Bnot(v *vm.VM) int {
	x := toUint32(v, 1, "bit32.bnot")
	v.Set(0, vm.NewInt(int64(^x)))
	return 1
}

// bit32.bor(...) - bitwise OR of all arguments. No args returns 0.
func bit32Bor(v *vm.VM) int {
	n := v.ArgCount()
	var result uint32 = 0
	for i := 1; i <= n; i++ {
		result |= toUint32(v, i, "bit32.bor")
	}
	v.Set(0, vm.NewInt(int64(result)))
	return 1
}

// bit32.btest(...) - returns true if bitwise AND of all arguments is non-zero.
func bit32Btest(v *vm.VM) int {
	n := v.ArgCount()
	var result uint32 = 0xFFFFFFFF
	for i := 1; i <= n; i++ {
		result &= toUint32(v, i, "bit32.btest")
	}
	v.Set(0, vm.NewBool(result != 0))
	return 1
}

// bit32.bxor(...) - bitwise XOR of all arguments. No args returns 0.
func bit32Bxor(v *vm.VM) int {
	n := v.ArgCount()
	var result uint32 = 0
	for i := 1; i <= n; i++ {
		result ^= toUint32(v, i, "bit32.bxor")
	}
	v.Set(0, vm.NewInt(int64(result)))
	return 1
}

// bit32.extract(n, field [, width]) - extract width bits starting at field
func bit32Extract(v *vm.VM) int {
	n := toUint32(v, 1, "bit32.extract")
	field := getInt(v, 2, "bit32.extract")
	width := int64(1)
	if v.ArgCount() >= 3 && !v.Get(3).IsNil() {
		width = getInt(v, 3, "bit32.extract")
	}

	if field < 0 || field > 31 {
		callerArgError(v, 2, "bit32.extract", "field cannot be negative or greater than 31")
	}
	if width < 1 || width > 32 {
		callerArgError(v, 3, "bit32.extract", "width must be positive and not greater than 32")
	}
	if field+width > 32 {
		callerArgError(v, 2, "bit32.extract", "trying to access non-existent bits")
	}

	var mask uint32
	if width == 32 {
		mask = 0xFFFFFFFF
	} else {
		mask = (1 << uint(width)) - 1
	}
	result := (n >> uint(field)) & mask

	v.Set(0, vm.NewInt(int64(result)))
	return 1
}

// bit32.replace(n, v, field [, width]) - replace width bits starting at field with v
func bit32Replace(v *vm.VM) int {
	n := toUint32(v, 1, "bit32.replace")
	rep := toUint32(v, 2, "bit32.replace")
	field := getInt(v, 3, "bit32.replace")
	width := int64(1)
	if v.ArgCount() >= 4 && !v.Get(4).IsNil() {
		width = getInt(v, 4, "bit32.replace")
	}

	if field < 0 || field > 31 {
		callerArgError(v, 3, "bit32.replace", "field cannot be negative or greater than 31")
	}
	if width < 1 || width > 32 {
		callerArgError(v, 4, "bit32.replace", "width must be positive and not greater than 32")
	}
	if field+width > 32 {
		callerArgError(v, 3, "bit32.replace", "trying to access non-existent bits")
	}

	var mask uint32
	if width == 32 {
		mask = 0xFFFFFFFF
	} else {
		mask = (1 << uint(width)) - 1
	}
	// Clear the bits in n, then OR in the replacement
	result := (n & ^(mask << uint(field))) | ((rep & mask) << uint(field))

	v.Set(0, vm.NewInt(int64(result)))
	return 1
}

// bit32.lrotate(x, disp) - left rotate
func bit32Lrotate(v *vm.VM) int {
	x := toUint32(v, 1, "bit32.lrotate")
	disp := getInt(v, 2, "bit32.lrotate")

	// Normalize displacement to [0, 31]
	disp = disp % 32
	if disp < 0 {
		disp += 32
	}

	result := (x << uint(disp)) | (x >> uint(32-disp))
	v.Set(0, vm.NewInt(int64(result)))
	return 1
}

// bit32.rrotate(x, disp) - right rotate
func bit32Rrotate(v *vm.VM) int {
	x := toUint32(v, 1, "bit32.rrotate")
	disp := getInt(v, 2, "bit32.rrotate")

	// Normalize displacement to [0, 31]
	disp = disp % 32
	if disp < 0 {
		disp += 32
	}

	result := (x >> uint(disp)) | (x << uint(32-disp))
	v.Set(0, vm.NewInt(int64(result)))
	return 1
}

// bit32.lshift(x, disp) - logical left shift
// Negative disp means right shift. Shifts >= 32 return 0.
func bit32Lshift(v *vm.VM) int {
	x := toUint32(v, 1, "bit32.lshift")
	disp := getInt(v, 2, "bit32.lshift")

	var result uint32
	if disp >= 0 {
		if disp >= 32 {
			result = 0
		} else {
			result = x << uint(disp)
		}
	} else {
		disp = -disp
		if disp >= 32 {
			result = 0
		} else {
			result = x >> uint(disp)
		}
	}

	v.Set(0, vm.NewInt(int64(result)))
	return 1
}

// bit32.rshift(x, disp) - logical right shift
// Negative disp means left shift. Shifts >= 32 return 0.
func bit32Rshift(v *vm.VM) int {
	x := toUint32(v, 1, "bit32.rshift")
	disp := getInt(v, 2, "bit32.rshift")

	var result uint32
	if disp >= 0 {
		if disp >= 32 {
			result = 0
		} else {
			result = x >> uint(disp)
		}
	} else {
		disp = -disp
		if disp >= 32 {
			result = 0
		} else {
			result = x << uint(disp)
		}
	}

	v.Set(0, vm.NewInt(int64(result)))
	return 1
}
