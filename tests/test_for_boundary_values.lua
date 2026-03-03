-- Test: nextvar.lua - Strange cases for numeric for (boundary values)
-- From: nextvar.lua
-- What: Tests numeric for loops with mininteger and maxinteger as boundaries and steps, including wrap-around and large step values.

do   -- testing other strange cases for numeric 'for'

  local function checkfor (from, to, step, t)
    local c = 0
    for i = from, to, step do
      c = c + 1
      assert(i == t[c])
    end
    assert(c == #t)
  end

  local maxi = math.maxinteger
  local mini = math.mininteger

  checkfor(mini, maxi, maxi, {mini, -1, maxi - 1})

  checkfor(mini, math.huge, maxi, {mini, -1, maxi - 1})

  checkfor(maxi, mini, mini, {maxi, -1})

  checkfor(maxi, mini, -maxi, {maxi, 0, -maxi})

  checkfor(maxi, -math.huge, mini, {maxi, -1})

  checkfor(maxi, mini, 1, {})
  checkfor(mini, maxi, -1, {})

  checkfor(maxi - 6, maxi, 3, {maxi - 6, maxi - 3, maxi})
  checkfor(mini + 4, mini, -2, {mini + 4, mini + 2, mini})

  local step = maxi // 10
  local c = mini
  for i = mini, maxi, step do
    assert(i == c)
    c = c + step
  end

  c = maxi
  for i = maxi, mini, -step do
    assert(i == c)
    c = c - step
  end

  checkfor(maxi, maxi, maxi, {maxi})
  checkfor(maxi, maxi, mini, {maxi})
  checkfor(mini, mini, maxi, {mini})
  checkfor(mini, mini, mini, {mini})
end
