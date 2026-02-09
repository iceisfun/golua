-- gsub should require 3 arguments
assert(not pcall(string.gsub, "x", "y"), "gsub needs 3 args")

-- gsub with non-integer limit should error
assert(not pcall(string.gsub, "x", "y", "z", "t"), "gsub bad limit type")

-- gsub with malformed pattern should error
assert(not pcall(string.gsub, "x", "%", "z"), "gsub malformed pattern")

-- gsub with invalid capture index should error
assert(not pcall(string.gsub, "xyz", "(x)", "%2"), "gsub invalid capture index")

-- gsub with boolean replacement should error
assert(not pcall(string.gsub, "xyz", "xyz", false), "gsub boolean repl")

-- gsub with table returning true should error (invalid replacement value)
assert(not pcall(string.gsub, "z", "z", {z = true}), "gsub table true value")

-- gsub function replacement error should propagate
assert(not pcall(string.gsub, "xyz", "xyz", function() error("baa") end), "gsub func error")
