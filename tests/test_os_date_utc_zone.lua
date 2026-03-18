-- Test os.date("!%Z", 0) returns "GMT" not "UTC"
-- Lua 5.4 uses C strftime which returns "GMT" for UTC timezone
local z = os.date("!%Z", 0)
assert(z == "GMT", "expected 'GMT', got '" .. tostring(z) .. "'")

print("OK")
