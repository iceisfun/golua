-- Test: pm.lua - Back-reference patterns
-- From: pm.lua
-- What: Tests pattern back-references (`%1`, `%2`) and position captures with string.match, including nested captures and equality matching.

do
  local function f1 (s, p)
    p = string.gsub(p, "%%([0-9])", function (s)
          return "%" .. (tonumber(s)+1)
         end)
    p = string.gsub(p, "^(^?)", "%1()", 1)
    p = string.gsub(p, "($?)$", "()%1", 1)
    local t = {string.match(s, p)}
    return string.sub(s, t[1], t[#t] - 1)
  end

  assert(f1('alo alx 123 b\0o b\0o', '(..*) %1') == "b\0o b\0o")
  assert(f1('axz123= 4= 4 34', '(.+)=(.*)=%2 %1') == '3= 4= 4 3')
  assert(f1('=======', '^(=*)=%1$') == '=======')
  assert(not string.match('==========', '^([=]*)=%1$'))
end
