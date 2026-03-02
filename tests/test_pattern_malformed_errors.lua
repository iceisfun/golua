-- Test: pm.lua - Malformed pattern errors
-- From: pm.lua
-- What: Tests that malformed patterns produce correct error messages for unfinished captures, invalid captures, unclosed brackets, incomplete `%b`, and missing `%f` set.

do
  local function malform (p, m)
    m = m or "malformed"
    local r, msg = pcall(string.find, "a", p)
    assert(not r and string.find(msg, m))
  end

  malform("(.", "unfinished capture")
  malform(".)", "invalid pattern capture")
  malform("[a")
  malform("[]")
  malform("[^]")
  malform("[a%]")
  malform("[a%")
  malform("%b")
  malform("%ba")
  malform("%")
  malform("%f", "missing")
end
