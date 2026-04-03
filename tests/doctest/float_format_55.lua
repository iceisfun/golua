-- Lua 5.5 float-to-string: shortest round-trip representation
-- Uses %.15g first, then %.17g if precision is lost.
-- Appends ".0" when result looks like an integer.

-- Basic integer-valued floats get ".0" suffix
print(tostring(1.0))
--> =1.0
print(tostring(0.0))
--> =0.0
print(tostring(-0.0))
--> =-0.0
print(tostring(100.0))
--> =100.0

-- Simple fractions: shortest representation
print(tostring(0.1))
--> =0.1
print(tostring(0.5))
--> =0.5
print(tostring(0.25))
--> =0.25
print(tostring(1.5))
--> =1.5

-- Repeating fractions: more digits than %.14g
print(tostring(1/3))
--> =0.33333333333333331
print(tostring(2/3))
--> =0.66666666666666663
print(tostring(math.pi))
--> =3.1415926535897931

-- Large numbers that are exact floats get ".0"
print(tostring(1e10))
--> =10000000000.0
print(tostring(2^53))
--> =9007199254740992.0

-- Scientific notation
print(tostring(1e100))
--> =1e+100
print(tostring(1e-10))
--> =1e-10

-- Special values
print(tostring(1/0))
--> =inf
print(tostring(-1/0))
--> =-inf
print(tostring(0/0))
--> =-nan

-- string.format("%s", ...) uses same format
print(string.format("%s", 1.0))
--> =1.0
print(string.format("%s", math.pi))
--> =3.1415926535897931

-- Concatenation uses same format
print(1.0 .. "")
--> =1.0
print(math.pi .. "")
--> =3.1415926535897931

-- io.write uses same format as tostring (Lua 5.5)
-- (tested separately in io_write_float_format.lua)

-- Explicit format specifiers in string.format are independent
print(string.format("%.14g", math.pi))
--> =3.1415926535898
print(string.format("%.14g", 1/3))
--> =0.33333333333333
