-- Test: pm.lua - Empty pattern and string.find basics
-- From: pm.lua
-- What: Tests string.find with empty patterns, null bytes, and basic substring searching including position offsets and anchored patterns.

do
  local function checkerror (msg, f, ...)
    local s, err = pcall(f, ...)
    assert(not s and string.find(err, msg))
  end


  local function f (s, p)
    local i,e = string.find(s, p)
    if i then return string.sub(s, i, e) end
  end

  local a,b = string.find('', '')    -- empty patterns are tricky
  assert(a == 1 and b == 0);
  a,b = string.find('alo', '')
  assert(a == 1 and b == 0)
  a,b = string.find('a\0o a\0o a\0o', 'a', 1)   -- first position
  assert(a == 1 and b == 1)
  a,b = string.find('a\0o a\0o a\0o', 'a\0o', 2)   -- starts in the midle
  assert(a == 5 and b == 7)
  a,b = string.find('a\0o a\0o a\0o', 'a\0o', 9)   -- starts in the midle
  assert(a == 9 and b == 11)
  a,b = string.find('a\0a\0a\0a\0\0ab', '\0ab', 2);  -- finds at the end
  assert(a == 9 and b == 11);
  a,b = string.find('a\0a\0a\0a\0\0ab', 'b')    -- last position
  assert(a == 11 and b == 11)
  assert(not string.find('a\0a\0a\0a\0\0ab', 'b\0'))   -- check ending
  assert(not string.find('', '\0'))
  assert(string.find('alo123alo', '12') == 4)
  assert(not string.find('alo123alo', '^12'))
end
