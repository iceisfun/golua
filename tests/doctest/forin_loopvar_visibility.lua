-- For-in loop variables should NOT be visible via debug.getlocal
-- during the iterator function call. They should only appear inside
-- the loop body.

local saw_k = false
local function myiter(t, k)
    -- Check if 'k' loop variable is visible at caller level
    local i = 1
    while true do
        local name = debug.getlocal(2, i)
        if not name then break end
        if name == "k" or name == "v" then saw_k = true end
        i = i + 1
    end
    k = (k or 0) + 1
    if k <= 2 then return k, t[k] end
end

for k, v in myiter, {"a", "b"}, nil do
end

print(saw_k)
--> =false
