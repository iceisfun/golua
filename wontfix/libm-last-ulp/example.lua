-- Last-ULP differences in transcendental / irrational math functions.
--
-- golua evaluates these with Go's `math` package; reference Lua uses the C
-- standard library (libm). Both are correctly rounded to within ~1 ULP, but the
-- last bit of the mantissa can differ for the same input. This is inherent to
-- using two different (both conforming) math libraries.

print(string.format("%.17g", 2.5 ^ 131))
--> golua:    1.3494013367335074e+52
--> lua5.5.0: 1.3494013367335069e+52   (differs in the last ~1 ULP)

-- The same class affects sin/cos/tan/asin/acos/atan/exp/log/sqrt(non-exact)/...
-- for some inputs. Many inputs agree (sqrt(2) happens to); the divergence is
-- input-specific, not a blanket offset.
print(string.format("%.17g", math.sqrt(2)))   --> both: 1.4142135623730951

print(2 ^ 53)        --> both: 9007199254740992.0  (exact powers agree)
print(math.sqrt(4))  --> both: 2.0                  (exact roots agree)
