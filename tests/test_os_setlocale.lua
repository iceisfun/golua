-- Test os.setlocale (stub)

-- Query current locale (nil arg) returns a non-empty locale string
local loc = os.setlocale(nil)
assert(type(loc) == "string" and #loc > 0)
loc = os.setlocale()
assert(type(loc) == "string" and #loc > 0)

-- Set to "C" returns "C"
assert(os.setlocale("C") == "C")

-- Set to "" uses environment locale and returns a non-empty locale string
loc = os.setlocale("")
assert(type(loc) == "string" and #loc > 0)

-- Set to unsupported locale returns nil
assert(os.setlocale("en_US.UTF-8") == nil)
assert(os.setlocale("POSIX") == nil)

-- With category argument (all categories return "C" for "C" locale)
assert(os.setlocale("C", "all") == "C")
assert(os.setlocale("C", "collate") == "C")
assert(os.setlocale("C", "ctype") == "C")
assert(os.setlocale("C", "monetary") == "C")
assert(os.setlocale("C", "numeric") == "C")
assert(os.setlocale("C", "time") == "C")

-- Query with category returns a non-empty locale string
loc = os.setlocale(nil, "all")
assert(type(loc) == "string" and #loc > 0)

print("PASS")
