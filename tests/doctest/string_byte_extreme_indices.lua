-- string.byte with extreme out-of-range indices must clamp to an empty range
-- and return no values, not overflow into a bogus "C stack overflow". The raw
-- index math (end - start + 1) can wrap when j = math.mininteger, so the
-- empty-range case must be detected before computing the count. Matches Lua 5.5.

print(pcall(string.byte, "hello", 255, -2^63))
--> =true

print(pcall(string.byte, "", 255, math.mininteger))
--> =true

-- inverted explicit range yields nothing
print(string.byte("hello", 3, 2))
--> =

-- huge upper bound clamps to string length
print(string.byte("hello", 1, math.maxinteger))
--> =104	101	108	108	111

-- huge negative lower bound clamps to start of string
print(string.byte("hello", math.mininteger, math.maxinteger))
--> =104	101	108	108	111
