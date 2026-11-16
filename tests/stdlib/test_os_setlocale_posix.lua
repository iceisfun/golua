-- Test: os.setlocale treats "POSIX" as the C locale
-- What: the POSIX name for the C locale is accepted and reported as "C"

assert(os.setlocale("C") == "C")
assert(os.setlocale("POSIX") == "C")
assert(os.setlocale("POSIX", "numeric") == "C")
assert(os.setlocale() == "C")
assert(os.setlocale("no_such_locale_xx") == nil)
