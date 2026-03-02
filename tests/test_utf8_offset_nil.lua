-- Test: utf8.offset out of range returns nil
-- From: utf8.lua
-- What: Tests that utf8.offset returns nil for out-of-range offsets.

do
assert(not utf8.offset("alo", 5))
assert(not utf8.offset("alo", -4))
end
