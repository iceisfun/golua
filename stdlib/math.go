package stdlib

import (
	"math"
	"math/rand"
	"time"

	"github.com/iceisfun/golua/vm"
)

func openMath(v *vm.VM) {
	m := vm.NewEmptyTable()

	// Constants
	m.SetString("pi", vm.NewFloat(math.Pi))
	m.SetString("huge", vm.NewFloat(math.Inf(1)))
	m.SetString("maxinteger", vm.NewInt(math.MaxInt64))
	m.SetString("mininteger", vm.NewInt(math.MinInt64))

	// Functions
	m.SetString("abs", vm.NewNativeFunc(mathAbs))
	m.SetString("acos", vm.NewNativeFunc(mathAcos))
	m.SetString("asin", vm.NewNativeFunc(mathAsin))
	m.SetString("atan", vm.NewNativeFunc(mathAtan))
	m.SetString("ceil", vm.NewNativeFunc(mathCeil))
	m.SetString("cos", vm.NewNativeFunc(mathCos))
	m.SetString("deg", vm.NewNativeFunc(mathDeg))
	m.SetString("exp", vm.NewNativeFunc(mathExp))
	m.SetString("floor", vm.NewNativeFunc(mathFloor))
	m.SetString("fmod", vm.NewNativeFunc(mathFmod))
	m.SetString("log", vm.NewNativeFunc(mathLog))
	m.SetString("max", vm.NewNativeFunc(mathMax))
	m.SetString("min", vm.NewNativeFunc(mathMin))
	m.SetString("modf", vm.NewNativeFunc(mathModf))
	m.SetString("rad", vm.NewNativeFunc(mathRad))
	m.SetString("random", vm.NewNativeFunc(mathRandom))
	m.SetString("randomseed", vm.NewNativeFunc(mathRandomseed))
	m.SetString("sin", vm.NewNativeFunc(mathSin))
	m.SetString("sqrt", vm.NewNativeFunc(mathSqrt))
	m.SetString("tan", vm.NewNativeFunc(mathTan))
	m.SetString("tointeger", vm.NewNativeFunc(mathTointeger))
	m.SetString("type", vm.NewNativeFunc(mathType))
	m.SetString("ult", vm.NewNativeFunc(mathUlt))

	v.SetGlobal("math", vm.NewTable(m))
}

func mathAbs(v *vm.VM) int {
	n := getNumber(v, 1, "abs")
	if v.Get(1).IsInt() {
		i := v.Get(1).AsInt()
		if i < 0 {
			v.Set(0, vm.NewInt(-i))
		} else {
			v.Set(0, vm.NewInt(i))
		}
	} else {
		v.Set(0, vm.NewFloat(math.Abs(n)))
	}
	return 1
}

func mathAcos(v *vm.VM) int {
	n := getNumber(v, 1, "acos")
	v.Set(0, vm.NewFloat(math.Acos(n)))
	return 1
}

func mathAsin(v *vm.VM) int {
	n := getNumber(v, 1, "asin")
	v.Set(0, vm.NewFloat(math.Asin(n)))
	return 1
}

func mathAtan(v *vm.VM) int {
	y := getNumber(v, 1, "atan")
	x := 1.0
	if !v.Get(2).IsNil() {
		x = getNumber(v, 2, "atan")
	}
	v.Set(0, vm.NewFloat(math.Atan2(y, x)))
	return 1
}

func mathCeil(v *vm.VM) int {
	n := getNumber(v, 1, "ceil")
	v.Set(0, vm.NewInt(int64(math.Ceil(n))))
	return 1
}

func mathCos(v *vm.VM) int {
	n := getNumber(v, 1, "cos")
	v.Set(0, vm.NewFloat(math.Cos(n)))
	return 1
}

func mathDeg(v *vm.VM) int {
	n := getNumber(v, 1, "deg")
	v.Set(0, vm.NewFloat(n * 180 / math.Pi))
	return 1
}

func mathExp(v *vm.VM) int {
	n := getNumber(v, 1, "exp")
	v.Set(0, vm.NewFloat(math.Exp(n)))
	return 1
}

func mathFloor(v *vm.VM) int {
	n := getNumber(v, 1, "floor")
	v.Set(0, vm.NewInt(int64(math.Floor(n))))
	return 1
}

func mathFmod(v *vm.VM) int {
	x := getNumber(v, 1, "fmod")
	y := getNumber(v, 2, "fmod")
	v.Set(0, vm.NewFloat(math.Mod(x, y)))
	return 1
}

func mathLog(v *vm.VM) int {
	x := getNumber(v, 1, "log")
	if v.Get(2).IsNil() {
		v.Set(0, vm.NewFloat(math.Log(x)))
	} else {
		base := getNumber(v, 2, "log")
		v.Set(0, vm.NewFloat(math.Log(x)/math.Log(base)))
	}
	return 1
}

func mathMax(v *vm.VM) int {
	n := v.ArgCount()
	if n == 0 {
		panic("bad argument #1 to 'max' (number expected, got no value)")
	}

	maxVal := v.Get(1)
	if !maxVal.IsNumber() {
		panic("bad argument #1 to 'max' (number expected)")
	}

	for i := 2; i <= n; i++ {
		cur := v.Get(i)
		if !cur.IsNumber() {
			panic("bad argument #1 to 'max' (number expected)")
		}
		if lt, _ := maxVal.LessThan(cur); lt {
			maxVal = cur
		}
	}

	v.Set(0, maxVal)
	return 1
}

func mathMin(v *vm.VM) int {
	n := v.ArgCount()
	if n == 0 {
		panic("bad argument #1 to 'min' (number expected, got no value)")
	}

	minVal := v.Get(1)
	if !minVal.IsNumber() {
		panic("bad argument #1 to 'min' (number expected)")
	}

	for i := 2; i <= n; i++ {
		cur := v.Get(i)
		if !cur.IsNumber() {
			panic("bad argument #1 to 'min' (number expected)")
		}
		if lt, _ := cur.LessThan(minVal); lt {
			minVal = cur
		}
	}

	v.Set(0, minVal)
	return 1
}

func mathModf(v *vm.VM) int {
	n := getNumber(v, 1, "modf")
	i, f := math.Modf(n)
	v.Set(0, vm.NewInt(int64(i)))
	v.Set(1, vm.NewFloat(f))
	return 2
}

func mathRad(v *vm.VM) int {
	n := getNumber(v, 1, "rad")
	v.Set(0, vm.NewFloat(n * math.Pi / 180))
	return 1
}

func mathRandom(v *vm.VM) int {
	n := v.ArgCount()

	switch n {
	case 0:
		// random() -> [0, 1)
		v.Set(0, vm.NewFloat(rand.Float64()))
	case 1:
		// random(n) -> [1, n]
		upper := getInt(v, 1, "random")
		if upper < 1 {
			panic("bad argument #1 to 'random' (interval is empty)")
		}
		v.Set(0, vm.NewInt(randRange(1, upper)))
	default:
		// random(m, n) -> [m, n]
		lower := getInt(v, 1, "random")
		upper := getInt(v, 2, "random")
		if lower > upper {
			panic("bad argument #2 to 'random' (interval is empty)")
		}
		v.Set(0, vm.NewInt(randRange(lower, upper)))
	}
	return 1
}

// randRange returns a random int64 in [lower, upper] (inclusive).
// Uses uint64 arithmetic to avoid overflow when the range spans
// the full 64-bit integer space (e.g., [MinInt64, MaxInt64]).
func randRange(lower, upper int64) int64 {
	r := uint64(upper) - uint64(lower)
	if r == 0 {
		return lower
	}
	if r == math.MaxUint64 {
		return int64(rand.Uint64())
	}
	return lower + int64(rand.Uint64()%(r+1))
}

func mathRandomseed(v *vm.VM) int {
	n := v.ArgCount()
	var seed1, seed2 int64

	switch {
	case n == 0:
		// No arguments: seed from system entropy
		seed1 = time.Now().UnixNano()
		seed2 = seed1 >> 16
	case n == 1:
		// One argument: use it as seed
		seed1 = getInt(v, 1, "randomseed")
		seed2 = 0
	default:
		// Two arguments: combine both
		seed1 = getInt(v, 1, "randomseed")
		seed2 = getInt(v, 2, "randomseed")
	}

	// Combine seed1 and seed2 into a single seed for Go's RNG
	combined := seed1 ^ (seed2 * 6364136223846793005)
	rand.Seed(combined)

	// Return the two seed state values per Lua 5.4
	v.Set(0, vm.NewInt(seed1))
	v.Set(1, vm.NewInt(seed2))
	return 2
}

func mathSin(v *vm.VM) int {
	n := getNumber(v, 1, "sin")
	v.Set(0, vm.NewFloat(math.Sin(n)))
	return 1
}

func mathSqrt(v *vm.VM) int {
	n := getNumber(v, 1, "sqrt")
	v.Set(0, vm.NewFloat(math.Sqrt(n)))
	return 1
}

func mathTan(v *vm.VM) int {
	n := getNumber(v, 1, "tan")
	v.Set(0, vm.NewFloat(math.Tan(n)))
	return 1
}

func mathTointeger(v *vm.VM) int {
	val := v.Get(1)
	if !val.IsNumber() {
		v.Set(0, vm.Nil)
		return 1
	}
	if i, ok := val.ToInt(); ok {
		v.Set(0, vm.NewInt(i))
	} else {
		v.Set(0, vm.Nil)
	}
	return 1
}

func mathType(v *vm.VM) int {
	val := v.Get(1)
	if !val.IsNumber() {
		v.Set(0, vm.Nil)
	} else if val.IsInt() {
		v.Set(0, vm.NewString("integer"))
	} else {
		v.Set(0, vm.NewString("float"))
	}
	return 1
}

func mathUlt(v *vm.VM) int {
	m := getInt(v, 1, "ult")
	n := getInt(v, 2, "ult")
	v.Set(0, vm.NewBool(uint64(m) < uint64(n)))
	return 1
}

func getNumber(v *vm.VM, idx int, fname string) float64 {
	val := v.Get(idx)
	if n, ok := val.ToNumber(); ok {
		return n
	}
	panic("bad argument #1 to '" + fname + "' (number expected)")
}
