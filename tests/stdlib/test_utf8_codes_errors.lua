-- Test: Errors in utf8.codes
-- From: utf8.lua
-- What: Tests that iterating with utf8.codes over invalid UTF-8 raises errors,
-- and that calling the iteration function with out-of-range positions returns nil.

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

  -- Lua 5.4 stops iteration at stray continuation bytes.
  do
    local t = {}
    for p, c in utf8.codes("in\x80valid") do
      t[#t+1] = p
      t[#t+1] = c
    end
    assert(#t == 4 and t[1] == 1 and t[2] == string.byte("i") and t[3] == 2 and t[4] == string.byte("n"))
  end
  do
    local f, s = utf8.codes("\xbfinvalid")
    assert(f(s, 0) == nil)
  end
  do
    local t = {}
    for p, c in utf8.codes("αλφ\xBFα") do
      t[#t+1] = p
      t[#t+1] = c
    end
    assert(#t == 6 and t[1] == 1 and t[3] == 3 and t[5] == 5)
  end

  local f = utf8.codes("")
  assert(f("", 2) == nil)
  assert(f("", -1) == nil)
  assert(f("", math.mininteger) == nil)

  local g, gs = utf8.codes("a")
  assert(g(gs, -1) == nil)
end
