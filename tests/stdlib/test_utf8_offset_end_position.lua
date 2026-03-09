-- utf8.offset with n=0 accepts i=#s+1 and returns #s+1.

do
  local s = string.char(127, 131, 86, 80, 236)
  assert(utf8.offset(s, 0, #s + 1) == #s + 1)

  local ascii = "abc"
  assert(utf8.offset(ascii, 0, #ascii + 1) == #ascii + 1)

  local empty = ""
  assert(utf8.offset(empty, 0, 1) == 1)
end
