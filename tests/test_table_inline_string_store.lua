-- test_table_inline_string_store:
-- Tables hold their first 8 string keys in an inline store (no Go map); the
-- 9th distinct string key migrates the whole set to the map. This exercises
-- the inline store, the overflow boundary, deletion/revival, weak mode, and
-- that pairs() still visits every key exactly once across both modes.
-- (Iteration *order* is covered separately by vm/table_test.go's TestNext*.)

local function count(t) local n = 0 for _ in pairs(t) do n = n + 1 end return n end

-- fill across the 8->9 overflow boundary; every key must read back
local t = {}
for i = 1, 20 do t["k" .. i] = i * 10 end
assert(count(t) == 20, "count after overflow: " .. count(t))
for i = 1, 20 do assert(t["k" .. i] == i * 10, "k" .. i) end

-- exactly 8 keys stays inline; all readable
local e = {}
for i = 1, 8 do e["f" .. i] = i end
assert(count(e) == 8 and e.f1 == 1 and e.f8 == 8, "inline-8")

-- delete then re-add — inline mode (tombstone revive)
local d = { a = 1, b = 2, c = 3 }
d.b = nil
assert(d.b == nil and count(d) == 2, "inline delete")
d.b = 22
assert(d.b == 22 and count(d) == 3, "inline revive")

-- delete then re-add — migrated (map) mode
local m = {}
for i = 1, 12 do m["x" .. i] = i end
m.x5 = nil
assert(m.x5 == nil and count(m) == 11, "map delete")
m.x5 = 555
assert(m.x5 == 555 and count(m) == 12, "map revive")

-- updating an existing key never changes the count
local u = { p = 1, q = 2 }
u.p = 100; u.q = 200; u.p = 101
assert(u.p == 101 and u.q == 200 and count(u) == 2, "update")

-- pairs() visits every surviving key exactly once across the overflow boundary
local seen = {}
for k, v in pairs(t) do
  assert(seen[k] == nil, "duplicate key in pairs: " .. tostring(k))
  seen[k] = v
end
for i = 1, 20 do assert(seen["k" .. i] == i * 10, "pairs missed k" .. i) end

-- nil a key mid-iteration: no key is visited twice
local it = {}
for i = 1, 10 do it["i" .. i] = i end
local visits = {}
for k in pairs(it) do
  visits[k] = (visits[k] or 0) + 1
  if k == "i3" then it.i7 = nil end
end
for k, c in pairs(visits) do assert(c == 1, "visited twice: " .. k) end

-- string keys remain distinct from equal-looking integer keys
local sn = {}
sn["1"] = "str"; sn[1] = "int"
assert(sn["1"] == "str" and sn[1] == "int", "string vs int key")

-- mixed array + string keys
local mix = { 10, 20, 30, name = "lua", ver = 5 }
assert(#mix == 3 and mix[1] == 10 and mix.name == "lua" and count(mix) == 5, "mixed")

-- string keys survive a weak-mode round trip (metatable __mode)
local w = { aa = 1, bb = 2, cc = 3 }
setmetatable(w, { __mode = "k" })
assert(w.aa == 1 and w.cc == 3 and count(w) == 3, "weak round trip")

print("ok")
