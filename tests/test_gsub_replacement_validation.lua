-- Bug: string.gsub does not validate % escapes in replacement strings.
-- lua5.4: string.gsub("abc", "a", "%") errors "invalid use of '%'"
-- golua: returns "%bc" (no error)
-- Also: string.gsub with negative n should treat as 0 replacements.

-- Test 1: % at end of replacement string should error
local ok, err = pcall(string.gsub, "abc", "a", "%")
assert(not ok, "expected error for trailing % in replacement, got: " .. tostring(ok))
assert(err:find("invalid use of"), "wrong error: " .. tostring(err))

-- Test 2: %x (invalid capture reference) should error
local ok2, err2 = pcall(string.gsub, "abc", "a", "%x")
assert(not ok2, "expected error for %x in replacement, got: " .. tostring(ok2))

-- Test 3: negative n should mean 0 replacements
local result, count = string.gsub("aaa", "a", "b", -1)
assert(result == "aaa", "expected 'aaa' with n=-1, got '" .. result .. "'")
assert(count == 0, "expected count=0 with n=-1, got " .. count)

-- Test 4: n=0 should mean 0 replacements (baseline)
local r0, c0 = string.gsub("aaa", "a", "b", 0)
assert(r0 == "aaa", "expected 'aaa' with n=0, got '" .. r0 .. "'")
assert(c0 == 0, "expected count=0 with n=0, got " .. c0)

print("PASS")
