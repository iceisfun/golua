-- Test: goto.lua - Repeat label in different function
-- From: goto.lua
-- What: Tests that labels with the same name can be used in different functions without conflict

do
  local function foo ()
    local a = {}
    goto l3
    ::l1:: a[#a + 1] = 1; goto l2;
    ::l2:: a[#a + 1] = 2; goto l5;
    ::l3::
    ::l3a:: a[#a + 1] = 3; goto l1;
    ::l4:: a[#a + 1] = 4; goto l6;
    ::l5:: a[#a + 1] = 5; goto l4;
    ::l6:: assert(a[1] == 3 and a[2] == 1 and a[3] == 2 and
                a[4] == 5 and a[5] == 4)
    if not a[6] then a[6] = true; goto l3a end
  end

  ::l6:: foo()
end
