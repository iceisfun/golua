-- Test: pm.lua - string.gmatch tests
-- From: pm.lua
-- What: Tests string.gmatch iterator for position captures, word extraction, repeated character detection, and key=value pair extraction.

do
  local a = 0
  for i in string.gmatch('abcde', '()') do assert(i == a+1); a=i end
  assert(a==6)

  local t = {n=0}
  for w in string.gmatch("first second word", "%w+") do
        t.n=t.n+1; t[t.n] = w
  end
  assert(t[1] == "first" and t[2] == "second" and t[3] == "word")

  t = {3, 6, 9}
  for i in string.gmatch ("xuxx uu ppar r", "()(.)%2") do
    assert(i == table.remove(t, 1))
  end
  assert(#t == 0)

  t = {}
  for i,j in string.gmatch("13 14 10 = 11, 15= 16, 22=23", "(%d+)%s*=%s*(%d+)") do
    t[tonumber(i)] = tonumber(j)
  end
  a = 0
  for k,v in pairs(t) do assert(k+1 == v+0); a=a+1 end
  assert(a == 3)
end
