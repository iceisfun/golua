-- string.match with malformed pattern should error
assert(not pcall(string.match, "x", "%"), "match malformed pattern should error")

-- string.gmatch with malformed pattern should error at iteration time
assert(not pcall(function()
    for w in string.gmatch("x", "%") do end
end), "gmatch malformed pattern should error")
