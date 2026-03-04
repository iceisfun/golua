-- Test: pm.lua - string.gsub error checking
-- From: pm.lua
-- What: Tests that string.gsub produces correct errors for invalid replacement values, invalid capture indices, and invalid `%` usage.

do
  local function checkerror (msg, f, ...)
    local s, err = pcall(f, ...)
    assert(not s and string.find(err, msg))
  end

  checkerror("invalid replacement value %(a table%)",
              string.gsub, "alo", ".", {a = {}})
  checkerror("invalid capture index %%2", string.gsub, "alo", ".", "%2")
  checkerror("invalid capture index %%0", string.gsub, "alo", "(%0)", "a")
  checkerror("invalid capture index %%1", string.gsub, "alo", "(%1)", "a")
  checkerror("invalid use of '%%'", string.gsub, "alo", ".", "%x")
end
