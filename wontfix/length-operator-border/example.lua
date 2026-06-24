-- The # (length) operator on a table with an interior hole.
--
-- For a table that is NOT a proper sequence (it has a nil "hole" in the middle),
-- Lua defines #t as ANY "border": an index n where t[n] ~= nil and t[n+1] == nil.
-- Such a table can have several valid borders; which one # returns is
-- implementation-defined. golua and reference Lua may pick different (both
-- valid) borders.

local t = {}
for i = 1, 100 do t[i] = i end
t[50] = nil          -- punch a hole; t is no longer a sequence

print(#t)
--> golua:    49    (t[49] ~= nil, t[50] == nil  -> 49 is a border)
--> lua5.5.0: 100   (t[100] ~= nil, t[101] == nil -> 100 is a border)
-- BOTH answers satisfy the definition of a border.

-- For a proper sequence (no holes) # is well-defined and both agree:
local s = {10, 20, 30, 40}
print(#s)            --> both: 4
