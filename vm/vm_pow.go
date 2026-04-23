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
// Special values (zero, NaN, Inf, negative subnormal) fall through to
// math.Pow, whose edge-case behavior (sign of zero/NaN, integer y on
// negative base, etc.) already matches Lua/libm expectations.
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
	mantBits := math.Float64bits(x) & ((1 << 52) - 1)
	m := float64(mantBits)
	return math.Pow(m, y) * math.Exp2(-1074.0*y)
}
