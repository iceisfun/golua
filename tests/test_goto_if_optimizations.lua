-- Test: goto.lua - If-goto optimizations
-- From: goto.lua
-- What: Tests goto within if-elseif-else chains with various branch paths

do
  local function testG (a)
    if a == 1 then
      goto l1
      error("should never be here!")
    elseif a == 2 then goto l2
    elseif a == 3 then goto l3
    elseif a == 4 then
      goto l1
      error("should never be here!")
      ::l1:: a = a + 1
    else
      goto l4
      ::l4a:: a = a * 2; goto l4b
      error("should never be here!")
      ::l4:: goto l4a
      error("should never be here!")
      ::l4b::
    end
    do return a end
    ::l2:: do return "2" end
    ::l3:: do return "3" end
    ::l1:: return "1"
  end

  assert(testG(1) == "1")
  assert(testG(2) == "2")
  assert(testG(3) == "3")
  assert(testG(4) == 5)
  assert(testG(5) == 10)
end
