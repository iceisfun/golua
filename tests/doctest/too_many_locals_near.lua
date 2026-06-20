-- Lua 5.5 checks the MAXVARS / register limits for a local declaration inside
-- adjustlocalvars / luaK_reserveregs, which run AFTER the whole statement
-- (names + attribs + '=' explist) has been parsed. The reported line and
-- "near '<token>'" are therefore those of the token that follows the statement
-- (the lookahead) — never the '=' or '<' inside it — and a trailing inlined
-- <const> compile-time constant is excluded from the limit.

-- Helper: build n separate "local vK = K" lines.
local function nlocals(n)
  local t = {}
  for i = 1, n do t[i] = "local v" .. i .. " = " .. i end
  return table.concat(t, "\n")
end

-- 201st plain local, followed by another statement: near the next token.
do
  local _, err = load(nlocals(200) .. "\nlocal y = 5\nprint(1)")
  print(err) --> ~:202: too many local variables \(limit is 200\) in main function near 'print'
end

-- 201st plain local at end of chunk: near <eof>.
do
  local _, err = load(nlocals(201))
  print(err) --> ~:201: too many local variables \(limit is 200\) in main function near <eof>
end

-- A trailing inlined <const> (literal initializer) is excluded from the limit,
-- so 200 locals + one such const compiles cleanly.
do
  local f = load(nlocals(200) .. "\nlocal k <const> = 1")
  print(type(f)) --> =function
end

-- Numeric for: when the visible control variable overflows, the limit is
-- reported after 'do' (here, near the body's first statement).
do
  local _, err = load(nlocals(198) .. "\nfor a = 1, 2 do local q = 1 end")
  print(err) --> ~:199: too many local variables \(limit is 200\) in main function near 'local'
end

-- Generic for: when the 3 internal state variables alone overflow, the limit is
-- reported at 'do'.
do
  local _, err = load(nlocals(198) .. "\nfor a, b, c in f do end")
  print(err) --> ~:199: too many local variables \(limit is 200\) in main function near 'do'
end

-- Declaring more locals in one statement than the register file (255) holds
-- reports the register limit first (registers are reserved before the MAXVARS
-- check), matching the reference compiler.
do
  local names = {}
  for i = 1, 300 do names[i] = "a" .. i end
  local code = "local " .. table.concat(names, ", ") .. " = 1\n"
  local _, err = load(code)
  print(err) --> ~too many registers \(limit is 255\)
end

print("PASS") --> =PASS
