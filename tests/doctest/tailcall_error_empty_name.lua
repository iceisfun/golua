-- A tail call of a nil value names the callee the same way a regular call
-- does. The empty string is a legitimate field or global name, so the
-- varinfo must be kept for it too. Matches reference Lua 5.5.

local t = {}

print(pcall(function() return t['']() end))
--> ~^false\t.*: attempt to call a nil value \(field ''\)$

print(pcall(function() return _ENV['']() end))
--> ~^false\t.*: attempt to call a nil value \(global ''\)$

print(pcall(function() t['']() end))
--> ~^false\t.*: attempt to call a nil value \(field ''\)$

print(pcall(function() return t.x() end))
--> ~^false\t.*: attempt to call a nil value \(field 'x'\)$
