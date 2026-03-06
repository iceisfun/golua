-- for-in TBC variable name should be '(for state)' not '?'
local ok, err = pcall(function()
    for v in (function() return nil end), nil, nil, 42 do end
end)
assert(not ok)
assert(err:find("%(for state%)"), "expected '(for state)' in: " .. tostring(err))
assert(not err:find("'%?'"), "should not contain '?' in: " .. tostring(err))

print("PASS")
