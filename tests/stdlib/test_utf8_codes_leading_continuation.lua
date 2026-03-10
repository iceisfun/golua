-- utf8.codes stops at stray continuation bytes instead of raising errors.

-- Leading continuation bytes → immediate nil
do
  local s = string.char(0xB2, 0xBD, 0xB8, 0x2A, 0x09, 0x5E)
  local f, str = utf8.codes(s)
  assert(f(str, 0) == nil, "leading continuation should stop iteration")
end

-- Stray continuation after valid codepoint → stop after earlier values
do
  local s = string.char(0x94, 0x7E, 0x69, 0xE5)
  local f, str = utf8.codes(s)
  assert(f(str, 0) == nil)
end

-- Trailing continuation byte after valid ASCII → stop after the ASCII byte
do
  local s = string.char(0x61, 0xBA)
  local out = {}
  for p, c in utf8.codes(s) do
    out[#out+1] = p
    out[#out+1] = c
  end
  assert(#out == 2 and out[1] == 1 and out[2] == string.byte("a"))
end
