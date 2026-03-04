-- Test: nextvar.lua - String-to-number conversion in numeric for
-- From: nextvar.lua
-- What: Tests that numeric for loop parameters accept string values that convert to numbers.

do
  local a = 0; for i="10","1","-2" do a=a+1 end; assert(a==5)
end
