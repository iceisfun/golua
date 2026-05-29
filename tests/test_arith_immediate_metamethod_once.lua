-- Regression: arithmetic with a small immediate operand (compiled to OP_ADDI,
-- which the compiler also uses for `x - smallconst` rewritten to `x + (-n)`)
-- must invoke the __add/__sub metamethod EXACTLY ONCE. Previously OP_ADDI
-- handled the metamethod itself AND the follow-up OP_MMBINI re-invoked it,
-- doubling any side effects (the result was masked because the last call won).

local function count(mm, build)
  local c = 0
  local t = setmetatable({}, {[mm] = function(a, b) c = c + 1; return 0 end})
  build(t)
  return c
end

-- __add via the immediate forms: t + n and (commutative) n + t.
assert(count("__add", function(t) local _ = t + 2 end) == 1, "t + 2 must call __add once")
assert(count("__add", function(t) local _ = 2 + t end) == 1, "2 + t must call __add once")

-- __sub: t - n is rewritten by the compiler to t + (-n) => OP_ADDI/TM_SUB.
assert(count("__sub", function(t) local _ = t - 2 end) == 1, "t - 2 must call __sub once")

-- Larger constants use OP_ADDK (no immediate) — must also fire once.
assert(count("__add", function(t) local _ = t + 1000000 end) == 1, "t + big must call __add once")

-- Bitwise immediates legitimately rely on OP_MMBINI (SHLI/SHRI/BANDK do not
-- pre-handle their metamethod) — must still fire exactly once.
assert(count("__shl", function(t) local _ = t << 2 end) == 1, "t << 2 must call __shl once")
assert(count("__band", function(t) local _ = t & 2 end) == 1, "t & 2 must call __band once")

-- The computed result must remain correct.
local v = setmetatable({n = 10}, {__add = function(a, b)
  local function num(x) return type(x) == "table" and x.n or x end
  return num(a) + num(b)
end})
assert((v + 2) == 12 and (2 + v) == 12, "metamethod result must be correct")

print("PASS")
