-- Test: Errors in utf8.codes
-- From: utf8.lua
-- What: Tests that iterating with utf8.codes over invalid UTF-8 raises errors, and that calling the iteration function with out-of-range positions returns nil.

do
  local function checkerror (msg, f, ...)
    local s, err = pcall(f, ...)
    assert(not s and string.find(err, msg))
  end

  local function errorcodes (s)
    checkerror("invalid UTF%-8 code",
      function ()
        for c in utf8.codes(s) do assert(c) end
      end)
  end
  errorcodes("ab\xff")
  errorcodes("\u{110000}")
  errorcodes("in\x80valid")
  errorcodes("\xbfinvalid")
  errorcodes("αλφ\xBFα")

  local f = utf8.codes("")
  assert(f("", 2) == nil)
  assert(f("", -1) == nil)
  assert(f("", math.mininteger) == nil)
end
