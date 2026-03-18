-- Test os.setlocale initial state and tracking
-- Lua 5.4 starts with locale "C" (set by luaL_openlibs)

-- Query current locale (no arg) - should be "C" initially
local cur = os.setlocale()
assert(cur == "C", "initial locale should be 'C', got '" .. tostring(cur) .. "'")

-- Set to "C" explicitly
local r = os.setlocale("C")
assert(r == "C", "setlocale('C') should return 'C', got '" .. tostring(r) .. "'")

-- Query again
local cur2 = os.setlocale()
assert(cur2 == "C", "locale after setlocale('C') should be 'C', got '" .. tostring(cur2) .. "'")

-- Unsupported locale returns nil
local r2 = os.setlocale("fr_FR")
assert(r2 == nil, "unsupported locale should return nil, got '" .. tostring(r2) .. "'")

-- After failed set, locale should remain unchanged
local cur3 = os.setlocale()
assert(cur3 == "C", "locale should remain 'C' after failed set, got '" .. tostring(cur3) .. "'")

print("OK")
