-- string.find with self-referencing back-reference should error
-- %1 inside capture 1 is an unfinished back-reference
local ok, err = pcall(string.find, "xxxx", "(xx%1)")
assert(not ok, "find with unfinished back-reference should error")
