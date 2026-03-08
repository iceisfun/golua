-- Test os.setlocale (stub)

-- Query current locale (nil arg) returns "C"
assert(os.setlocale(nil) == "C")
assert(os.setlocale() == "C")

-- Set to "C" returns "C"
assert(os.setlocale("C") == "C")

-- Set to "" (query) returns "C"
assert(os.setlocale("") == "C")

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

-- Query with category
assert(os.setlocale(nil, "all") == "C")

print("PASS")
