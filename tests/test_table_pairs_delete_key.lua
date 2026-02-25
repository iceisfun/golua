-- Tests that deleting a key during pairs() iteration does not terminate the loop early.
-- Lua 5.4 reference manual: "You may modify existing fields. In particular, you may clear existing fields."

local t = {}
t.A = 1
t.B = 2
t.C = 3
t.D = 4

local count = 0
for k, v in pairs(t) do
    count = count + 1
    if k == "B" then
        t.B = nil -- clear the current key
    end
end

if count ~= 4 then
    error("pairs iteration terminated early after deleting a key; expected 4 visited keys, got " .. tostring(count))
end
