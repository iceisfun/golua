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
	return libmNaNSignFix(x, y, powWithSubnormalFixImpl(x, y))
}

func powWithSubnormalFixImpl(x, y float64) float64 {
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

// libmNaNSignFix adjusts the sign bit of a NaN result from math.Pow to
// match libm's pow on Linux x86-64 (which lua5.4 / lua5.5 expose). The
// IEEE-754 / C99 specs leave the sign of a NaN result unspecified, so
// Go's math.Pow and libm pow disagree on the choice in several cases.
// Lua 5.4 and 5.5 both call libm's pow, so for parity we mimic libm here.
//
// Empirical libm rules (Linux x86-64) for the sign of a NaN result:
//
//   - For NaN^y with finite non-NaN y, y != 0:
//     y == 1 or y == -1   → result is +nan (sign flipped from -nan input)
//     odd integer y       → result sign = !signbit(x)
//     otherwise           → result sign =  signbit(x)
//   - For x^NaN with finite non-NaN x, x != 1:
//     result sign = signbit(y)
//   - For NaN^NaN:
//     result sign = signbit(x)
//   - For finite x < 0 ^ non-integer y (Go gives +nan in some cases):
//     result sign = - (-nan)
//
// Other inputs (y == 0, x == 1) are not NaN and pass through unchanged.
func libmNaNSignFix(x, y, r float64) float64 {
	if !math.IsNaN(r) {
		return r
	}
	xIsNaN := math.IsNaN(x)
	yIsNaN := math.IsNaN(y)
	switch {
	case xIsNaN && yIsNaN:
		return math.Copysign(r, signOf(x))
	case xIsNaN:
		// nan ^ y. Special libm cases.
		if y == 1 || y == -1 {
			return math.Copysign(r, +1)
		}
		if isOddInt(y) {
			return math.Copysign(r, -signOf(x))
		}
		return math.Copysign(r, signOf(x))
	case yIsNaN:
		// x ^ nan, x != 1 (else not NaN). Sign tracks y.
		return math.Copysign(r, signOf(y))
	default:
		// Finite x < 0 with non-integer y. libm yields -nan.
		return math.Copysign(r, -1)
	}
}

// signOf returns +1 or -1 based on the sign bit of f. Works for NaN.
func signOf(f float64) float64 {
	if math.Signbit(f) {
		return -1
	}
	return 1
}

// isOddInt reports whether y is an odd integer.
func isOddInt(y float64) bool {
	if math.IsNaN(y) || math.IsInf(y, 0) {
		return false
	}
	if math.Abs(y) >= (1 << 53) {
		// Beyond float64 integer precision; cannot be odd.
		return false
	}
	yi, yf := math.Modf(y)
	return yf == 0 && int64(yi)&1 == 1
}
