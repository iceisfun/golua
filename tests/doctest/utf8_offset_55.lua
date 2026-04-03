-- utf8.offset returns two values in Lua 5.5:
-- the byte position where the character starts, and the byte position
-- where it ends (position of the last byte of the character encoding).

-- ASCII strings: single-byte chars, second value equals first
print(utf8.offset("hello", 1))
--> =1	1
print(utf8.offset("hello", 2))
--> =2	2
print(utf8.offset("hello", 3))
--> =3	3
print(utf8.offset("hello", 5))
--> =5	5

-- Past-end sentinel: both values are #s+1
print(utf8.offset("hello", 6))
--> =6	6

-- Negative offsets on ASCII
print(utf8.offset("hello", -1))
--> =5	5
print(utf8.offset("hello", -5))
--> =1	1

-- Multi-byte: 2-byte characters (ç = C3 A7, í = C3 AD)
-- "açaí" bytes: a(1) ç(2-3) a(4) í(5-6)
print(utf8.offset("açaí", 1))
--> =1	1
print(utf8.offset("açaí", 2))
--> =2	3
print(utf8.offset("açaí", 3))
--> =4	4
print(utf8.offset("açaí", 4))
--> =5	6
-- Past-end sentinel
print(utf8.offset("açaí", 5))
--> =7	7

-- Negative offsets on multi-byte
print(utf8.offset("açaí", -1))
--> =5	6
print(utf8.offset("açaí", -2))
--> =4	4
print(utf8.offset("açaí", -3))
--> =2	3
print(utf8.offset("açaí", -4))
--> =1	1

-- Multi-byte: 3-byte characters (日本語)
-- "日本語" bytes: 日(1-3) 本(4-6) 語(7-9)
print(utf8.offset("日本語", 1))
--> =1	3
print(utf8.offset("日本語", 2))
--> =4	6
print(utf8.offset("日本語", 3))
--> =7	9
-- Past-end sentinel
print(utf8.offset("日本語", 4))
--> =10	10

-- Negative offsets on 3-byte characters
print(utf8.offset("日本語", -1))
--> =7	9
print(utf8.offset("日本語", -2))
--> =4	6
print(utf8.offset("日本語", -3))
--> =1	3

-- n=0 cases: returns start of current character and its last byte
print(utf8.offset("hello", 0))
--> =1	1
print(utf8.offset("hello", 0, 3))
--> =3	3

-- n=0 on multi-byte: snaps back to character start
-- ç occupies bytes 2-3; both byte positions should resolve to (2, 3)
print(utf8.offset("açaí", 0, 2))
--> =2	3
print(utf8.offset("açaí", 0, 3))
--> =2	3

-- n=0 on 3-byte char: 日 occupies bytes 1-3
print(utf8.offset("日本語", 0, 1))
--> =1	3
print(utf8.offset("日本語", 0, 2))
--> =1	3
print(utf8.offset("日本語", 0, 3))
--> =1	3

-- n=0 at past-end position
print(utf8.offset("abc", 0, 4))
--> =4	4

-- Empty string: offset 1 returns the past-end sentinel
print(utf8.offset("", 1))
--> =1	1

-- offset returns nil when target is unreachable
print(utf8.offset("abc", 5))
--> =nil
print(utf8.offset("abc", -4))
--> =nil

-- Explicit start position (3rd argument)
print(utf8.offset("hello", 1, 3))
--> =3	3
print(utf8.offset("hello", 2, 3))
--> =4	4

-- Missing continuation bytes: incomplete sequences
-- \xE0 is a 3-byte lead but has no continuation bytes
local p, e = utf8.offset("\xE0", 1)
print(p, e)
--> =1	1

-- \xE0\x9E is a 3-byte lead with only 1 continuation byte
local p, e = utf8.offset("\xE0\x9E", -1)
print(p, e)
--> =1	2

-- Mixed ASCII and multi-byte with explicit position
-- "日本語a-4" bytes: 日(1-3) 本(4-6) 語(7-9) a(10) -(11) 4(12)
print(utf8.offset("日本語a-4", 4))
--> =10	10
print(utf8.offset("日本語a-4", -1))
--> =12	12
print(utf8.offset("日本語a-4", -3))
--> =10	10
print(utf8.offset("日本語a-4", -4))
--> =7	9
