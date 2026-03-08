-- Test os.date strftime specifiers
-- Uses fixed Unix timestamps (UTC) for reproducibility

-- Known UTC timestamps:
-- 2024-06-15 14:30:45 UTC (Saturday) = 1718458245
-- 2024-01-01 00:00:00 UTC (Monday)   = 1704067200
-- 2024-12-31 23:59:59 UTC (Tuesday)  = 1735689599
-- 2024-02-29 12:00:00 UTC (Thursday) = 1709208000

local t1 = 1718461845  -- 2024-06-15 14:30:45 UTC, Saturday
local t2 = 1704067200  -- 2024-01-01 00:00:00 UTC, Monday
local t3 = 1735689599  -- 2024-12-31 23:59:59 UTC, Tuesday
local t4 = 1709208000  -- 2024-02-29 12:00:00 UTC, Thursday

-- Sanity check timestamps
assert(os.date("!%Y-%m-%d %H:%M:%S", t1) == "2024-06-15 14:30:45", "t1 sanity failed")
assert(os.date("!%Y-%m-%d %H:%M:%S", t2) == "2024-01-01 00:00:00", "t2 sanity failed")
assert(os.date("!%Y-%m-%d %H:%M:%S", t3) == "2024-12-31 23:59:59", "t3 sanity failed")
assert(os.date("!%Y-%m-%d %H:%M:%S", t4) == "2024-02-29 12:00:00", "t4 sanity failed")

-- %C - century
assert(os.date("!%C", t1) == "20", "%C failed: got " .. os.date("!%C", t1))

-- %D - equivalent to %m/%d/%y
assert(os.date("!%D", t1) == "06/15/24", "%D failed: got " .. os.date("!%D", t1))

-- %e - space-padded day
assert(os.date("!%e", t2) == " 1", "%e for day 1 failed: got '" .. os.date("!%e", t2) .. "'")
assert(os.date("!%e", t1) == "15", "%e for day 15 failed: got '" .. os.date("!%e", t1) .. "'")

-- %F - equivalent to %Y-%m-%d
assert(os.date("!%F", t1) == "2024-06-15", "%F failed: got " .. os.date("!%F", t1))

-- %g - ISO 8601 2-digit year
assert(os.date("!%g", t1) == "24", "%g failed: got " .. os.date("!%g", t1))

-- %G - ISO 8601 4-digit year
assert(os.date("!%G", t1) == "2024", "%G failed: got " .. os.date("!%G", t1))
-- Dec 31, 2024 is in ISO week 1 of 2025
assert(os.date("!%G", t3) == "2025", "%G for Dec 31 failed: got " .. os.date("!%G", t3))

-- %j - day of year (zero-padded to 3 digits)
assert(os.date("!%j", t2) == "001", "%j for Jan 1 failed: got " .. os.date("!%j", t2))
assert(os.date("!%j", t1) == "167", "%j for Jun 15 failed: got " .. os.date("!%j", t1))
assert(os.date("!%j", t3) == "366", "%j for Dec 31 failed: got " .. os.date("!%j", t3))

-- %n - literal newline
assert(os.date("!%n", t1) == "\n", "%n failed")

-- %r - equivalent to %I:%M:%S %p
assert(os.date("!%r", t1) == "02:30:45 PM", "%r failed: got " .. os.date("!%r", t1))
assert(os.date("!%r", t2) == "12:00:00 AM", "%r failed for midnight: got " .. os.date("!%r", t2))

-- %R - equivalent to %H:%M
assert(os.date("!%R", t1) == "14:30", "%R failed: got " .. os.date("!%R", t1))

-- %T - equivalent to %H:%M:%S
assert(os.date("!%T", t1) == "14:30:45", "%T failed: got " .. os.date("!%T", t1))

-- %t - literal tab
assert(os.date("!%t", t1) == "\t", "%t failed")

-- %u - ISO weekday (Monday=1, Sunday=7)
assert(os.date("!%u", t1) == "6", "%u for Saturday failed: got " .. os.date("!%u", t1))
assert(os.date("!%u", t2) == "1", "%u for Monday failed: got " .. os.date("!%u", t2))

-- %U - week of year (Sunday as first day)
assert(os.date("!%U", t2) == "00", "%U for Jan 1 (Mon) failed: got " .. os.date("!%U", t2))

-- %V - ISO 8601 week number
assert(os.date("!%V", t1) == "24", "%V failed: got " .. os.date("!%V", t1))
assert(os.date("!%V", t3) == "01", "%V for Dec 31 failed: got " .. os.date("!%V", t3))

-- %w - weekday number (Sunday=0)
assert(os.date("!%w", t1) == "6", "%w for Saturday failed: got " .. os.date("!%w", t1))
assert(os.date("!%w", t2) == "1", "%w for Monday failed: got " .. os.date("!%w", t2))

-- %W - week of year (Monday as first day)
assert(os.date("!%W", t2) == "01", "%W for Jan 1 (Mon) failed: got " .. os.date("!%W", t2))

-- Test compound specifiers mixed with other text
assert(os.date("!Date: %F Time: %T", t1) == "Date: 2024-06-15 Time: 14:30:45",
  "compound mixed failed")

-- Test type checking: boolean should error
local ok, err = pcall(os.date, true)
assert(not ok, "os.date(true) should error")
assert(string.find(err, "string expected, got boolean"),
  "os.date(true) error message wrong: " .. tostring(err))

local ok2, err2 = pcall(os.date, false)
assert(not ok2, "os.date(false) should error")
assert(string.find(err2, "string expected, got boolean"),
  "os.date(false) error message wrong: " .. tostring(err2))

-- Test number coercion: number should be converted to format string
-- os.date(12345) should use "12345" as format (no % specifiers, so literal)
local numResult = os.date(12345)
assert(numResult == "12345", "os.date(12345) should coerce number to string, got: " .. tostring(numResult))

print("PASS")
