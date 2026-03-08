-- Test: "too many registers" error includes "near" token

local function checkmessage(code, expectedmsg)
    local f, err = load(code)
    if f then
        local ok
        ok, err = pcall(f)
    end
    assert(err and string.find(err, expectedmsg, 1, true),
           "expected '" .. expectedmsg .. "' in: " .. tostring(err))
end

-- The error should include "near 'x'" for the expression that overflows
checkmessage("a = f(x" .. string.rep(",x", 260) .. ")", "too many registers")
checkmessage("a = f(x" .. string.rep(",x", 260) .. ")", "near 'x'")

print("OK")
