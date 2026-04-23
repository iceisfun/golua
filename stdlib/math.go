package stdlib

import (
	"math"
	"math/bits"
	"time"

	"github.com/iceisfun/golua/v2/vm"
)

// xoshiro256ss implements the xoshiro256** pseudo-random number generator
// used by Lua 5.4. This matches the exact algorithm and seeding behavior
// so that math.random output is identical to reference Lua.
type xoshiro256ss struct {
	s [4]uint64
}

// seed initializes the RNG state from two seed values, matching Lua 5.4's setseed().
func (x *xoshiro256ss) seed(n1, n2 int64) {
	x.s[0] = uint64(n1)
	x.s[1] = 0xff // avoid a zero state
	x.s[2] = uint64(n2)
	x.s[3] = 0
	// Discard initial values to spread the seed
	for i := 0; i < 16; i++ {
		x.next()
	}
}

// next generates the next random uint64 using xoshiro256**.
func (x *xoshiro256ss) next() uint64 {
	s0 := x.s[0]
	s1 := x.s[1]
	s2 := x.s[2] ^ s0
	s3 := x.s[3] ^ s1
	res := bits.RotateLeft64(s1*5, 7) * 9
	x.s[0] = s0 ^ s3
	x.s[1] = s1 ^ s2
	x.s[2] = s2 ^ (s1 << 17)
	x.s[3] = bits.RotateLeft64(s3, 45)
	return res
}

// float64 returns a random float64 in [0, 1) with 53 bits of precision,
// matching Lua 5.4's I2d conversion.
func (x *xoshiro256ss) float64() float64 {
	const figs = 53 // significant bits in a double
	r := x.next() >> (64 - figs)
	return float64(r) / float64(uint64(1)<<figs)
}

func openMath(v *vm.VM) {
	m := vm.NewEmptyTable()

	// Per-VM xoshiro256** random source (matches Lua 5.4 exactly)
	rng := &xoshiro256ss{}
	rng.seed(time.Now().UnixNano(), 0)
	atanVal := vm.NewNativeFunc(mathAtan)

	// Constants
	m.SetString("pi", vm.NewFloat(math.Pi))
	m.SetString("huge", vm.NewFloat(math.Inf(1)))
	m.SetString("maxinteger", vm.NewInt(math.MaxInt64))
	m.SetString("mininteger", vm.NewInt(math.MinInt64))

	// Functions
	m.SetString("abs", vm.NewNativeFunc(mathAbs))
	m.SetString("acos", vm.NewNativeFunc(mathAcos))
	m.SetString("asin", vm.NewNativeFunc(mathAsin))
	m.SetString("atan", atanVal)
	m.SetString("ceil", vm.NewNativeFunc(mathCeil))
	m.SetString("cos", vm.NewNativeFunc(mathCos))
	m.SetString("deg", vm.NewNativeFunc(mathDeg))
	m.SetString("exp", vm.NewNativeFunc(mathExp))
	m.SetString("floor", vm.NewNativeFunc(mathFloor))
	m.SetString("frexp", vm.NewNativeFunc(mathFrexp))
	m.SetString("fmod", vm.NewNativeFunc(mathFmod))
	m.SetString("ldexp", vm.NewNativeFunc(mathLdexp))
	m.SetString("log", vm.NewNativeFunc(mathLog))
	m.SetString("max", vm.NewNativeFunc(mathMax))
	m.SetString("min", vm.NewNativeFunc(mathMin))
	m.SetString("modf", vm.NewNativeFunc(mathModf))
	m.SetString("rad", vm.NewNativeFunc(mathRad))
	m.SetString("random", vm.NewNativeFunc(mathRandomClosure(rng)))
	m.SetString("randomseed", vm.NewNativeFunc(mathRandomseedClosure(rng)))
	m.SetString("sin", vm.NewNativeFunc(mathSin))
	m.SetString("sqrt", vm.NewNativeFunc(mathSqrt))
	m.SetString("tan", vm.NewNativeFunc(mathTan))
	m.SetString("tointeger", vm.NewNativeFunc(mathTointeger))
	m.SetString("type", vm.NewNativeFunc(mathType))
	m.SetString("ult", vm.NewNativeFunc(mathUlt))

	v.SetGlobal("math", vm.NewTable(m))
}

func mathAbs(v *vm.VM) int {
	n := getNumber(v, 1, "math.abs")
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
	n := getNumber(v, 1, "math.acos")
	result := math.Acos(n)
	if math.IsNaN(result) {
		if math.IsNaN(n) {
			result = math.Copysign(math.NaN(), -1)
		} else {
			result = math.NaN()
		}
	}
	v.Set(0, vm.NewFloat(result))
	return 1
}

func mathAsin(v *vm.VM) int {
	n := getNumber(v, 1, "math.asin")
	result := math.Asin(n)
	if math.IsNaN(result) {
		if math.IsNaN(n) {
			result = math.Copysign(math.NaN(), -1)
		} else {
			result = math.NaN()
		}
	}
	v.Set(0, vm.NewFloat(result))
	return 1
}

func mathAtan(v *vm.VM) int {
	y := getNumber(v, 1, "math.atan")
	x := 1.0
	if !v.Get(2).IsNil() {
		x = getNumber(v, 2, "math.atan")
	}
	result := math.Atan2(y, x)
	if math.IsNaN(result) {
		result = math.Copysign(result, -1)
	}
	v.Set(0, vm.NewFloat(result))
	return 1
}

func mathCeil(v *vm.VM) int {
	arg := v.Get(1)
	if arg.IsInt() {
		v.Set(0, arg)
		return 1
	}
	n := getNumber(v, 1, "math.ceil")
	f := math.Ceil(n)
	if !math.IsNaN(f) && !math.IsInf(f, 0) && f >= -9223372036854775808 && f < 9223372036854775808 {
		v.Set(0, vm.NewInt(int64(f)))
	} else {
		v.Set(0, vm.NewFloat(f))
	}
	return 1
}

func mathCos(v *vm.VM) int {
	n := getNumber(v, 1, "math.cos")
	result := math.Cos(n)
	if math.IsNaN(result) {
		result = math.Copysign(result, -1)
	}
	v.Set(0, vm.NewFloat(result))
	return 1
}

func mathCosh(v *vm.VM) int {
	n := getNumber(v, 1, "math.cosh")
	v.Set(0, vm.NewFloat(math.Cosh(n)))
	return 1
}

func mathDeg(v *vm.VM) int {
	n := getNumber(v, 1, "math.deg")
	v.Set(0, vm.NewFloat(n*180/math.Pi))
	return 1
}

func mathExp(v *vm.VM) int {
	n := getNumber(v, 1, "math.exp")
	v.Set(0, vm.NewFloat(math.Exp(n)))
	return 1
}

func mathFloor(v *vm.VM) int {
	arg := v.Get(1)
	if arg.IsInt() {
		v.Set(0, arg)
		return 1
	}
	n := getNumber(v, 1, "math.floor")
	f := math.Floor(n)
	if !math.IsNaN(f) && !math.IsInf(f, 0) && f >= -9223372036854775808 && f < 9223372036854775808 {
		v.Set(0, vm.NewInt(int64(f)))
	} else {
		v.Set(0, vm.NewFloat(f))
	}
	return 1
}

func mathFrexp(v *vm.VM) int {
	n := getNumber(v, 1, "math.frexp")
	if math.IsNaN(n) {
		v.Set(0, vm.NewFloat(math.Copysign(math.NaN(), -1)))
		v.Set(1, vm.NewInt(0))
		return 2
	}
	m, e := math.Frexp(n)
	v.Set(0, vm.NewFloat(m))
	v.Set(1, vm.NewInt(int64(e)))
	return 2
}

func mathFmod(v *vm.VM) int {
	v1 := v.Get(1)
	v2 := v.Get(2)
	y := getNumber(v, 2, "math.fmod")
	x := getNumber(v, 1, "math.fmod")
	if v1.IsInt() && v2.IsInt() {
		if y == 0 {
			callerArgError(v, 2, "math.fmod", "zero")
		}
		// Integer fmod: preserve integer type
		a := v1.AsInt()
		b := v2.AsInt()
		v.Set(0, vm.NewInt(a%b))
		return 1
	}
	result := math.Mod(x, y)
	// Go's math.Mod returns positive NaN; C's fmod returns negative NaN.
	if math.IsNaN(result) {
		result = math.Copysign(math.NaN(), -1)
	}
	v.Set(0, vm.NewFloat(result))
	return 1
}

func mathLog(v *vm.VM) int {
	x := getNumber(v, 1, "math.log")
	var result float64
	base10 := false
	if v.Get(2).IsNil() {
		result = math.Log(x)
	} else {
		base := getNumber(v, 2, "math.log")
		if base == 10.0 {
			base10 = true
			result = math.Log10(x)
		} else if base == 2.0 {
			result = math.Log2(x)
		} else {
			result = math.Log(x) / math.Log(base)
		}
	}
	if math.IsNaN(result) {
		if base10 {
			// Go's math.Log10 returns signed NaN for negative inputs;
			// glibc's log10 returns positive NaN. Canonicalize to match Lua 5.5.
			result = math.NaN()
		} else {
			// Go's math.Log returns positive NaN for negative inputs;
			// C's log returns negative NaN. Force negative NaN to match Lua 5.5
			// for the natural-log and non-10 base paths.
			result = math.Copysign(math.NaN(), -1)
		}
	}
	v.Set(0, vm.NewFloat(result))
	return 1
}

func mathLdexp(v *vm.VM) int {
	x := getNumber(v, 1, "math.ldexp")
	exp := getInt(v, 2, "math.ldexp")
	v.Set(0, vm.NewFloat(math.Ldexp(x, int(exp))))
	return 1
}

func mathLog10(v *vm.VM) int {
	x := getNumber(v, 1, "math.log10")
	v.Set(0, vm.NewFloat(math.Log10(x)))
	return 1
}

func mathMax(v *vm.VM) int {
	n := v.ArgCount()
	if n == 0 {
		callerArgError(v, 1, "math.max", "value expected")
	}

	maxVal := v.Get(1)

	for i := 2; i <= n; i++ {
		cur := v.Get(i)
		// Use Lua < operator (supports strings, __lt metamethods)
		lt, err := v.CompareLT(maxVal, cur)
		if err != nil {
			panic(err)
		}
		if lt {
			maxVal = cur
		}
	}

	v.Set(0, maxVal)
	return 1
}

func mathMin(v *vm.VM) int {
	n := v.ArgCount()
	if n == 0 {
		callerArgError(v, 1, "math.min", "value expected")
	}

	minVal := v.Get(1)

	for i := 2; i <= n; i++ {
		cur := v.Get(i)
		// Use Lua < operator (supports strings, __lt metamethods)
		lt, err := v.CompareLT(cur, minVal)
		if err != nil {
			panic(err)
		}
		if lt {
			minVal = cur
		}
	}

	v.Set(0, minVal)
	return 1
}

func mathModf(v *vm.VM) int {
	val := v.Get(1)
	// If the argument is already an integer, return it directly with 0.0 fraction.
	// This avoids precision loss from converting large integers (e.g. maxinteger)
	// to float64 and back.
	if val.IsInt() {
		v.Set(0, val)
		v.Set(1, vm.NewFloat(0))
		return 2
	}
	n := getNumber(v, 1, "math.modf")
	if math.IsInf(n, 0) {
		v.Set(0, vm.NewFloat(n))
		v.Set(1, vm.NewFloat(0))
		return 2
	}
	i, f := math.Modf(n)
	if f == 0 {
		f = 0 // normalize negative zero to positive zero
	}
	if !math.IsNaN(i) && i >= -9223372036854775808 && i < 9223372036854775808 {
		v.Set(0, vm.NewInt(int64(i)))
	} else {
		v.Set(0, vm.NewFloat(i))
	}
	v.Set(1, vm.NewFloat(f))
	return 2
}

func mathRad(v *vm.VM) int {
	n := getNumber(v, 1, "math.rad")
	v.Set(0, vm.NewFloat(n*math.Pi/180))
	return 1
}

func mathPow(v *vm.VM) int {
	x := getNumber(v, 1, "math.pow")
	y := getNumber(v, 2, "math.pow")
	v.Set(0, vm.NewFloat(vm.PowWithSubnormalFix(x, y)))
	return 1
}

func mathRandomClosure(rng *xoshiro256ss) vm.NativeFunc {
	return func(v *vm.VM) int {
		n := v.ArgCount()

		switch n {
		case 0:
			// random() -> [0, 1) with exactly 53 bits of randomness
			v.Set(0, vm.NewFloat(rng.float64()))
		case 1:
			// random(0) -> raw internal state as integer (Lua 5.4)
			// random(n) -> [1, n]
			upper := getInt(v, 1, "math.random")
			if upper == 0 {
				v.Set(0, vm.NewInt(int64(rng.next())))
			} else if upper < 1 {
				callerArgError(v, 1, "math.random", "interval is empty")
			} else {
				v.Set(0, vm.NewInt(xoshiroRange(rng, 1, upper)))
			}
		case 2:
			// random(m, n) -> [m, n]
			lower := getInt(v, 1, "math.random")
			upper := getInt(v, 2, "math.random")
			if lower > upper {
				callerArgError(v, 1, "math.random", "interval is empty")
			}
			v.Set(0, vm.NewInt(xoshiroRange(rng, lower, upper)))
		default:
			panic("wrong number of arguments")
		}
		return 1
	}
}

// xoshiroRange returns a random int64 in [lower, upper] (inclusive)
// using the xoshiro256** RNG, matching Lua 5.4's project() function.
func xoshiroRange(rng *xoshiro256ss, lower, upper int64) int64 {
	r := uint64(upper) - uint64(lower)
	if r == 0 {
		return lower
	}
	if r == math.MaxUint64 {
		// Full range: Lua 5.4 computes p = project(I2UInt(rv), MaxUint64)
		// which is identity, then returns (lua_Integer)(p + (lua_Unsigned)low).
		// All arithmetic is unsigned before the final cast to signed.
		ran := rng.next()
		return int64(ran + uint64(lower))
	}
	// Lua 5.4's project(): bitmask rejection sampling.
	// Uses the exact same algorithm as lmathlib.c for identical sequences.
	ran := rng.next()
	return lower + int64(project(ran, r, rng))
}

// project maps a random value into [0..n] using Lua 5.4's bitmask
// rejection sampling algorithm from lmathlib.c.
func project(ran, n uint64, rng *xoshiro256ss) uint64 {
	if (n & (n + 1)) == 0 {
		// n+1 is a power of 2: no bias with simple mask
		return ran & n
	}
	// Find the smallest (2^b - 1) >= n
	lim := n
	lim |= lim >> 1
	lim |= lim >> 2
	lim |= lim >> 4
	lim |= lim >> 8
	lim |= lim >> 16
	lim |= lim >> 32
	// Project ran into [0..lim], retry if > n
	ran &= lim
	for ran > n {
		ran = rng.next() & lim
	}
	return ran
}

func mathRandomseedClosure(rng *xoshiro256ss) vm.NativeFunc {
	return func(v *vm.VM) int {
		n := v.ArgCount()
		var seed1, seed2 int64

		switch {
		case n == 0:
			// No arguments: seed from system entropy
			seed1 = time.Now().UnixNano()
			seed2 = seed1 >> 16
		case n == 1:
			// One argument: use it as seed
			seed1 = getInt(v, 1, "math.randomseed")
			seed2 = 0
		default:
			// Two arguments: combine both
			seed1 = getInt(v, 1, "math.randomseed")
			seed2 = getInt(v, 2, "math.randomseed")
		}

		rng.seed(seed1, seed2)

		// Return the two seed state values per Lua 5.4
		v.Set(0, vm.NewInt(seed1))
		v.Set(1, vm.NewInt(seed2))
		return 2
	}
}

func mathSin(v *vm.VM) int {
	n := getNumber(v, 1, "math.sin")
	result := math.Sin(n)
	if math.IsNaN(result) {
		result = math.Copysign(result, -1)
	}
	v.Set(0, vm.NewFloat(result))
	return 1
}

func mathSinh(v *vm.VM) int {
	n := getNumber(v, 1, "math.sinh")
	v.Set(0, vm.NewFloat(math.Sinh(n)))
	return 1
}

func mathSqrt(v *vm.VM) int {
	n := getNumber(v, 1, "math.sqrt")
	v.Set(0, vm.NewFloat(math.Sqrt(n)))
	return 1
}

func mathTan(v *vm.VM) int {
	n := getNumber(v, 1, "math.tan")
	result := math.Tan(n)
	if math.IsNaN(result) {
		result = math.Copysign(result, -1)
	}
	v.Set(0, vm.NewFloat(result))
	return 1
}

func mathTanh(v *vm.VM) int {
	n := getNumber(v, 1, "math.tanh")
	v.Set(0, vm.NewFloat(math.Tanh(n)))
	return 1
}

func mathTointeger(v *vm.VM) int {
	if v.ArgCount() < 1 {
		callerArgError(v, 1, "math.tointeger", "value expected")
	}
	val := v.Get(1)
	if !val.IsNumber() && !val.IsString() {
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
	if v.ArgCount() < 1 {
		callerArgError(v, 1, "math.type", "value expected")
	}
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
	m := getInt(v, 1, "math.ult")
	n := getInt(v, 2, "math.ult")
	v.Set(0, vm.NewBool(uint64(m) < uint64(n)))
	return 1
}
