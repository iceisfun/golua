-- Signed hex strings ("-0xff", "+0xA") must coerce to integer, not float,
-- in arithmetic, string.format, and tonumber.

-- Arithmetic coercion preserves integer type
print("-0xff" + 0, math.type("-0xff" + 0))
--> =-255	integer
print("+0xff" + 0, math.type("+0xff" + 0))
--> =255	integer
print("-0xA" + 0, math.type("-0xA" + 0))
--> =-10	integer

-- string.format %d with signed hex strings
print(string.format("%d", "-0xff"))
--> =-255
print(string.format("%d", "+0xff"))
--> =255
print(string.format("%x", "-0x10"))
--> =fffffffffffffff0

-- string.format %f with signed hex strings
print(string.format("%f", "-0xff"))
--> =-255.000000

-- tonumber with signed hex
print(tonumber("-0xff"), math.type(tonumber("-0xff")))
--> =-255	integer
print(tonumber("+0xff"), math.type(tonumber("+0xff")))
--> =255	integer

-- Edge: overflow wraps
print("-0xFFFFFFFFFFFFFFFF" + 0)
--> =1
-- Edge: signed zero
print("+0x0" + 0, math.type("+0x0" + 0))
--> =0	integer
