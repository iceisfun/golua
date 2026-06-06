-- Lua 5.5 reserves a register for a "(vararg table)" local at index
-- NumParams whenever a function's parameter list contains "...". For a
-- named vararg ("... name") that register holds a real table; for a plain
-- "..." it holds a hidden nil. Either way the slot shifts debug.getlocal
-- numbering: the first body local lands one slot later than it did in
-- Lua 5.4. The main chunk is vararg but has no parameter list, so it
-- reserves no slot.

-- Plain "..." reserves a hidden "(vararg table)" slot at NumParams (here
-- register 2, the third slot): a(1) b(2) (vararg table)(3) x(4).
local function plain(a, b, ...)
  local x = 99
  local n3 = debug.getlocal(1, 3)
  local n4 = debug.getlocal(1, 4)
  return n3 .. "|" .. n4
end
print(plain(1, 2, 3))
--> =(vararg table)|x

-- The hidden slot reads back as nil (PF_VAHID — no table is materialized).
local function hidden(...)
  local _, v = debug.getlocal(1, 1)
  return tostring(v)
end
print(hidden(1, 2, 3))
--> =nil

-- The static-function form probes pc=0, where only the fixed parameters are
-- active; the vararg slot (registered after VARARGPREP) is not yet visible,
-- so debug.getlocal(f, NumParams+1) is nil.
local function probe(a, b, ...) local d end
print(tostring(debug.getlocal(probe, 1)),
      tostring(debug.getlocal(probe, 2)),
      tostring(debug.getlocal(probe, 3)))
--> =a	b	nil

-- The main chunk is vararg but declares no parameter list, so it reserves no
-- slot: its first declared local ("plain", above) sits at slot 1. A reserved
-- vararg slot would have pushed it to slot 2.
print((debug.getlocal(1, 1)))
--> =plain
