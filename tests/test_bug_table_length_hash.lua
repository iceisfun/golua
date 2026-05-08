-- Lua 5.5 # semantics for sparse-array constructors: walk forward from
-- index 1 returning the first contiguous run, or 0 when t[1] is nil.

-- Constructor with leading nil: t[1] is nil → # is 0.
assert(#{nil, "a"} == 0, "#{nil, 'a'} should be 0, got " .. #{nil, "a"})

-- Multiple leading nils: same.
assert(#{nil, nil, 3} == 0, "#{nil, nil, 3} should be 0, got " .. #{nil, nil, 3})

-- Interior nil: # stops at the first hole.
assert(#{1, nil, 3} == 1, "#{1, nil, 3} should be 1, got " .. #{1, nil, 3})

-- Guard: normal sequential tables.
assert(#{1, 2, 3} == 3)
assert(#{} == 0)
assert(#{"a"} == 1)

-- table.concat over an empty range produces "".
assert(table.concat({nil, "a"}, ",") == "",
  "table.concat({nil,'a'}) should be empty since #t==0")
