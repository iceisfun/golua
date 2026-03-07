-- Error messages from stdlib functions should include file:line: prefix
-- Bug: stdlib errors lacked the calling Lua frame's source location

-- math.abs
do
    local ok, err = pcall(function() math.abs() end)
    assert(not ok)
    assert(string.find(err, "^.+:%d+: "), "missing location prefix: " .. tostring(err))
end

-- string.find pattern error
do
    local ok, err = pcall(function() string.find("abc", "%") end)
    assert(not ok)
    assert(string.find(err, "^.+:%d+: "), "missing location prefix: " .. tostring(err))
end

-- table.sort invalid order
do
    local ok, err = pcall(function()
        table.sort({3,2,1,3,2,1}, function() return true end)
    end)
    assert(not ok)
    assert(string.find(err, "^.+:%d+: "), "missing location prefix: " .. tostring(err))
end

-- Direct call (no Lua frame above) - may not have prefix
do
    local ok, err = pcall(math.abs)
    assert(not ok)
    -- This is fine with or without prefix
end

