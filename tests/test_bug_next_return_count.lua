-- Bug #4: next() at end of iteration returns 2 values (nil, nil) instead of 1 (nil)

assert(select("#", next({})) == 1, "next on empty table should return 1 value")
assert(select("#", next({a=1}, "a")) == 1, "next past last key should return 1 value")

-- When there IS a next element, should return 2 values (key, value)
assert(select("#", next({a=1})) == 2, "next with element should return 2 values")
