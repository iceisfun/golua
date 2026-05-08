-- Lua 5.5 changed the # operator's algorithm: it walks forward from
-- index 1 (starting near asize/2 with a 4-step vicinity probe), so for
-- small sparse tables the result is the index of the first nil minus 1
-- (or 0 when t[1] is nil).  Verified against lua5.5.0.

assert(#{1, nil, 3} == 1)
assert(#{1, 2, nil, 4} == 2)
assert(#{nil, 2, 3} == 0)
assert(#{1, 2, 3, nil, 5, 6} == 3)
assert(#{1, nil, nil, nil, 5} == 1)

-- Table.sort no longer errors on a sparse table starting with nil because
-- the sort range is [1..#t] = [1..0] = empty.
do
  local ok = pcall(table.sort, {nil, 2, 3})
  assert(ok)
end

-- Common patterns: dense tables and trivial cases still work.
assert(#{} == 0)
assert(#{"a"} == 1)
assert(#{1, 2, 3} == 3)
assert(#{1, 2, 3, 4, 5, 6, 7, 8, 9, 10} == 10)

-- Trailing nil: # stops before the last hole.
assert(#{1, 2, 3, nil} == 3)
assert(#{nil, nil, nil, nil, 5, nil} == 0)
