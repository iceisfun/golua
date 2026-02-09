-- string.rep with negative count should return empty string (Lua 5.4)
assert(string.rep("xx", -1) == "", "rep with negative count should return empty")
assert(string.rep("abc", 0) == "", "rep with zero count should return empty")
assert(string.rep("x", 3) == "xxx", "rep basic")
