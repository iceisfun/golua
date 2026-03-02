-- Test: Error indication in utf8.len
-- From: utf8.lua
-- What: Tests that utf8.len returns nil and the position of the first invalid byte for malformed UTF-8 sequences.

do
  local function check (s, p)
    local a, b = utf8.len(s)
    assert(not a and b == p)
  end
  check("abc\xE3def", 4)
  check("\xF4\x9F\xBF", 1)
  check("\xF4\x9F\xBF\xBF", 1)
  check("汉字\x80", #("汉字") + 1)
  check("\x80hello", 1)
  check("hel\x80lo", 4)
  check("汉字\xBF", #("汉字") + 1)
  check("\xBFhello", 1)
  check("hel\xBFlo", 4)
end
