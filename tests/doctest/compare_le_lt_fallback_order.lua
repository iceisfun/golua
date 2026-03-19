-- When <= falls back to __lt, Lua 5.4 picks the right operand's __lt first
-- and then swaps the call arguments to evaluate not (b < a).

local a = setmetatable({ id = "a" }, {
    __lt = function(x, y)
        print("a_lt", x.id, y.id)
        return true
    end,
})

local b = setmetatable({ id = "b" }, {
    __lt = function(x, y)
        print("b_lt", x.id, y.id)
        return false
    end,
})

print("a<=b", a <= b)
--> =b_lt	b	a
--> =a<=b	true
print("b<=a", b <= a)
--> =a_lt	a	b
--> =b<=a	false
