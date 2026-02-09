-- gsub with position captures should substitute the numeric positions
local r = string.gsub("xyz", "()y()", "%1-%2")
assert(r == "x2-3z", "gsub position capture: expected 'x2-3z', got: " .. r)
