-- Exercise doctest.set_timeout
-- Lowering the timeout should work; raising should fail.

---------------------------------------------------------------------
-- set_timeout with valid lower value
---------------------------------------------------------------------
doctest.set_timeout(5)
print("timeout lowered")
--> =timeout lowered

---------------------------------------------------------------------
-- set_timeout rejects non-positive values
---------------------------------------------------------------------
local err1 = doctest.expect_error(function()
    doctest.set_timeout(0)
end)
print(type(err1))
--> =string

local err2 = doctest.expect_error(function()
    doctest.set_timeout(-1)
end)
print(type(err2))
--> =string

---------------------------------------------------------------------
-- set_timeout rejects nil
---------------------------------------------------------------------
local err3 = doctest.expect_error(function()
    doctest.set_timeout(nil)
end)
print(type(err3))
--> =string

print("timeout tests passed")
--> =timeout tests passed
