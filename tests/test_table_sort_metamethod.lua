-- Bug: table.sort default comparator ignores __lt metamethod.
-- Uses raw Value.LessThan() instead of vm.lessThan() which respects metamethods.

-- Custom objects that sort via __lt metamethod
local mt = {
  __lt = function(a, b) return a.val < b.val end,
  __le = function(a, b) return a.val <= b.val end,
}

local function obj(v)
  return setmetatable({val = v}, mt)
end

local t = {obj(3), obj(1), obj(2)}
table.sort(t)
assert(t[1].val == 1, "expected 1, got " .. t[1].val)
assert(t[2].val == 2, "expected 2, got " .. t[2].val)
assert(t[3].val == 3, "expected 3, got " .. t[3].val)

-- Sort with custom comparator should still work too
local t2 = {obj(3), obj(1), obj(2)}
table.sort(t2, function(a, b) return a.val > b.val end)
assert(t2[1].val == 3, "reverse sort: expected 3, got " .. t2[1].val)
assert(t2[2].val == 2, "reverse sort: expected 2, got " .. t2[2].val)
assert(t2[3].val == 1, "reverse sort: expected 1, got " .. t2[3].val)

