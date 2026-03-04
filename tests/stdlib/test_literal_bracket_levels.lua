-- Test: Long bracket strings with various bracket levels
-- From: literals.lua
-- What: Tests parsing of long bracket strings with different bracket levels and nested bracket patterns.

do
  local a = [==[]=]==]
  assert(a == "]=")

  a = [==[[===[[=[]]=][====[]]===]===]==]
  assert(a == "[===[[=[]]=][====[]]===]===")

  a = [====[[===[[=[]]=][====[]]===]===]====]
  assert(a == "[===[[=[]]=][====[]]===]===")

  a = [=[]]]]]]]]]=]
  assert(a == "]]]]]]]]")
end
