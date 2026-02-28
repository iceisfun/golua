-- Exercise doctest helper functions
-- Each helper is tested for both success and expected failure behavior.

---------------------------------------------------------------------
-- doctest.assert
---------------------------------------------------------------------
doctest.assert(true)
doctest.assert(1)
doctest.assert("non-empty")
doctest.assert({})
print("assert passed")
--> =assert passed

---------------------------------------------------------------------
-- doctest.expect_equal
---------------------------------------------------------------------
doctest.expect_equal(1, 1)
doctest.expect_equal("hello", "hello")
doctest.expect_equal(1, 1.0)
doctest.expect_equal(nil, nil)
doctest.expect_equal(true, true)
print("expect_equal passed")
--> =expect_equal passed

---------------------------------------------------------------------
-- doctest.expect_type
---------------------------------------------------------------------
doctest.expect_type(nil, "nil")
doctest.expect_type(true, "boolean")
doctest.expect_type(42, "number")
doctest.expect_type(3.14, "number")
doctest.expect_type("hi", "string")
doctest.expect_type({}, "table")
doctest.expect_type(print, "function")
print("expect_type passed")
--> =expect_type passed

---------------------------------------------------------------------
-- doctest.expect_error
---------------------------------------------------------------------
local err = doctest.expect_error(function()
    error("boom")
end)
print(type(err))
--> =string

local err2 = doctest.expect_error(function()
    error(42)
end)
print(type(err2))
--> =number

local err3 = doctest.expect_error(function()
    error({msg = "table error"})
end)
print(type(err3))
--> =table

---------------------------------------------------------------------
-- doctest.fail caught by expect_error
---------------------------------------------------------------------
local err4 = doctest.expect_error(function()
    doctest.fail("intentional failure")
end)
print(type(err4))
--> =string

---------------------------------------------------------------------
-- doctest.assert failure caught by expect_error
---------------------------------------------------------------------
local err5 = doctest.expect_error(function()
    doctest.assert(false, "should fail")
end)
print(type(err5))
--> =string

---------------------------------------------------------------------
-- doctest table exists and is a table
---------------------------------------------------------------------
print(type(doctest))
--> =table

print("all helpers passed")
--> =all helpers passed
