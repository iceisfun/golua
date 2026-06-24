-- Signed-zero through compile-time constant folding.
--
-- All operands are literal constants, so both implementations fold the
-- arithmetic at compile time. golua's folder is IEEE 754 correct: (-0.0) - 0.0
-- is -0.0. Reference Lua's constant folder produces +0.0 here.
--
-- The parentheses matter: they keep the expression a pure constant the folder
-- can see. A runtime value (e.g. local x = -0.0; x - 0) agrees on both.

print(string.format("%.17g", ((-0.0)) - (0)))
--> golua:     -0
--> lua5.5.0:   0

-- Negation of a constant zero, same root cause:
print(string.format("%.17g", -(0.0)))
--> golua:     -0
--> lua5.5.0:   0   (reference folds -(0.0) to +0.0)

-- Runtime (non-folded) signed zero is identical on both:
local z = 0.0
print(string.format("%.17g", -z))
--> both:      -0
