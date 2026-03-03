-- Test: pm.lua - Character set patterns (bracket expressions)
-- From: pm.lua
-- What: Tests character sets in brackets including ranges, complements, special characters, `%Z`, and `.` matching the full 0-255 byte range.

do
  local function range (i, j)
    if i <= j then
      return i, range(i+1, j)
    end
  end

  local abc = string.char(range(0, 127)) .. string.char(range(128, 255));

  assert(string.len(abc) == 256)

  local function strset (p)
    local res = {s=''}
    string.gsub(abc, p, function (c) res.s = res.s .. c end)
    return res.s
  end;

  assert(string.len(strset('[\200-\210]')) == 11)

  assert(strset('[a-z]') == "abcdefghijklmnopqrstuvwxyz")
  assert(strset('[a-z%d]') == strset('[%da-uu-z]'))
  assert(strset('[a-]') == "-a")
  assert(strset('[^%W]') == strset('[%w]'))
  assert(strset('[]%%]') == '%]')
  assert(strset('[a%-z]') == '-az')
  assert(strset('[%^%[%-a%]%-b]') == '-[]^ab')
  assert(strset('%Z') == strset('[\1-\255]'))
  assert(strset('.') == strset('[\1-\255%z]'))
end
