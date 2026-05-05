-- broken_fuzz_pairs_next_after_trailing_nil:
-- pairs() / next() infinite loop after table grown to cap > len, with an
-- integer-hash key K placed past array len but within array cap.
--
-- BROKEN: After a table is sized so cap(t.array) > len(t.array) (easy to hit
-- via constructor with explicit trailing nil, or via table.create(narr) with
-- sparse subsequent fills), calling next(t, K) for an integer-hash key K
-- with len < K <= cap returns (K, V) instead of advancing past K.
-- Consequence: `for k,v in pairs(t) do ... end` loops forever on K.
--
-- Suspect site: vm/table.go, Table.Next around the cap(t.array) recovery
-- branch. The branch returns t.firstLiveHashEntry() — when the queried key
-- IS itself the first live hash entry, that's the same key, looping. The
-- nextHashAfter(hashKey(key)) path immediately below already handles the
-- "queried key was a deleted array slot" case correctly.
--
-- Reference (lua5.5.0 and lua 5.4.8 both):
--   next(t, 8) -> nil
--   for k,v in pairs(t) terminates after iterating all keys
--
-- golua today:
--   next(t, 8) -> 8, 200  (returns same key; pairs spins forever)
--
-- Discovered: differential fuzz 2026-05-04 (table-library wave-1 agent).
-- Production-relevant: any user who calls table.create(narr) and then
-- populates sparse integer keys can hit this.

-- Variant 1: trailing-nil constructor + sparse keys past array len
local t = {1, 2, 3, nil}
t[5] = 99
t[6] = 100
t[8] = 200

assert(next(t, 8) == nil,
  "next(t, 8) must return nil when 8 is the last live key; got " ..
  tostring(({next(t, 8)})[1]))

local count = 0
for k, v in pairs(t) do
  count = count + 1
  assert(count <= 10, "pairs(t) iterated more than 10 times — infinite loop")
end
assert(count == 6, "expected 6 keys, got " .. count)

print("ok")
