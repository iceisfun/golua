package vm

import "math"

// PowWithSubnormalFix computes x^y with a rescaling fix for positive
// subnormal (denormal) bases.
//
// Go's math.Pow uses Exp(yf*Log(x)) * x**yi, which loses many orders of
// magnitude of precision when Log(x) is evaluated deep in the denormal
// range. libm's pow (which Lua 5.5 uses) avoids this via extended
// precision. To match libm for positive subnormals we decompose
//
//	x = m * 2^-1074
//
// where m is the raw mantissa (an integer in [1, 2^52-1]) and thus a
// perfectly normal float. Then
//
//	x^y = m^y * 2^(-1074*y)
//
// Both factors are computed in the normal float64 range.
//
// Special values (zero, NaN, Inf, negative subnormal, infinite y) fall
// through to math.Pow, whose edge-case behavior (sign of zero/NaN, integer
// y on negative base, x^±Inf for 0<x<1, etc.) already matches Lua/libm
// expectations.
//
// For finite y the rescale formula is evaluated as
//
//	m^y * 2^(-1074*y)
//
// using math.Ldexp for the integer part of the rescale exponent so that
// neither factor overflows independently when |y| is large. If m^y itself
// over- or underflows (which happens for |y| >> 1 with m far from 1) the
// rescale fix cannot help and we fall back to math.Pow, whose macroscopic
// answer for subnormal bases at extreme y is already correct.
func PowWithSubnormalFix(x, y float64) float64 {
	if x == 0 || math.IsNaN(x) || math.IsInf(x, 0) || math.IsNaN(y) {
		return math.Pow(x, y)
	}
	// smallestNormal = 2^-1022
	const smallestNormal = 2.2250738585072014e-308
	if math.Abs(x) >= smallestNormal {
		return math.Pow(x, y)
	}
	if x < 0 {
		// Negative subnormal: math.Pow already returns NaN for non-integer y
		// and handles integer y via x**yi; both match libm.
		return math.Pow(x, y)
	}
	if math.IsInf(y, 0) {
		// y = ±Inf with 0 < x < 1: math.Pow gives 0 / +Inf, matching libm.
		return math.Pow(x, y)
	}
	mantBits := math.Float64bits(x) & ((1 << 52) - 1)
	m := float64(mantBits)
	my := math.Pow(m, y)
	if my == 0 || math.IsInf(my, 0) || math.IsNaN(my) {
		// m^y degenerated (huge |y| pushed it past float range). Defer to
		// math.Pow, which produces the correct ±0 / ±Inf limit for
		// subnormal x at extreme y.
		return math.Pow(x, y)
	}
	// Rescale exponent e = -1074*y. Split into integer and fractional parts
	// so we can use math.Ldexp (which combines the exponent with the
	// mantissa exactly, saturating to ±0/±Inf only at the float64 limits).
	e := -1074.0 * y
	ei, ef := math.Modf(e)
	// Clamp ei to int range; Ldexp saturates correctly for huge |iexp|.
	const maxLdexp = 1 << 30
	var iexp int
	switch {
	case ei > maxLdexp:
		iexp = maxLdexp
	case ei < -maxLdexp:
		iexp = -maxLdexp
	default:
		iexp = int(ei)
	}
	return math.Ldexp(my, iexp) * math.Exp2(ef)
}
