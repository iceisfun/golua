-- test_table_sort_comparator.lua
-- table.sort must honor a user comparator for ordering.

local values = {1, 2, 3, 4, 5}

-- sort descending using comparator
local function desc(a, b)
    return a > b
end

table.sort(values, desc)

for i = 1, #values - 1 do
    assert(values[i] >= values[i + 1], string.format("expected descending order, got %s", table.concat(values, ",")))
end

assert(values[1] == 5 and values[#values] == 1, string.format("expected full reverse, got %s", table.concat(values, ",")))
