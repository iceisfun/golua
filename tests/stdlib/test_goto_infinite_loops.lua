-- Test: goto.lua - Compiling infinite loops
-- From: goto.lua
-- What: Tests that the compiler can handle infinite loops formed by goto statements

do
  goto escape
  ::a:: goto a
  ::b:: goto c
  ::c:: goto b
end
::escape::
