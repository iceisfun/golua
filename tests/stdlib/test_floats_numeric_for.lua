-- Test: nextvar.lua - Floats in numeric for
-- From: nextvar.lua
-- What: Tests numeric for loops with float boundaries and steps, verifying integer vs float iteration counts.

do
  local a
  -- integer count
  a = 0; for i=1, 1, 1 do a=a+1 end; assert(a==1)
  a = 0; for i=10000, 1e4, -1 do a=a+1 end; assert(a==1)
  a = 0; for i=1, 0.99999, 1 do a=a+1 end; assert(a==0)
  a = 0; for i=9999, 1e4, -1 do a=a+1 end; assert(a==0)
  a = 0; for i=1, 0.99999, -1 do a=a+1 end; assert(a==1)

  -- float count
  a = 0; for i=0, 0.999999999, 0.1 do a=a+1 end; assert(a==10)
  a = 0; for i=1.0, 1, 1 do a=a+1 end; assert(a==1)
  a = 0; for i=-1.5, -1.5, 1 do a=a+1 end; assert(a==1)
  a = 0; for i=1e6, 1e6, -1 do a=a+1 end; assert(a==1)
  a = 0; for i=1.0, 0.99999, 1 do a=a+1 end; assert(a==0)
  a = 0; for i=99999, 1e5, -1.0 do a=a+1 end; assert(a==0)
  a = 0; for i=1.0, 0.99999, -1 do a=a+1 end; assert(a==1)
end
