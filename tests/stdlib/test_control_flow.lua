-- Test: constructs.lua - Control flow (if/while/repeat/for)
-- From: constructs.lua
-- What: Tests various control flow constructs including nested loops, break, repeat-until, for loops

do
  for i=1,1000 do break; end;
  local n=100; local i=3; local t = {}; local a=nil
  while not a do
    a=0; for i=1,n do for i=i,1,-1 do a=a+1; t[i]=1; end; end;
  end
  assert(a == n*(n+1)/2 and i==3);

  function f(b)
    local x = 1;
    repeat
      local a;
      if b==1 then local b=1; x=10; break
      elseif b==2 then x=20; break;
      elseif b==3 then x=30;
      else local a,b,c,d=math.sin(1); x=x+1;
      end
    until x>=12;
    return x;
  end;
  assert(f(1) == 10 and f(2) == 20 and f(3) == 30 and f(4)==12)
end
