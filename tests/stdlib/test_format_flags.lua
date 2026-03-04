-- Test: string.format flags (ISO C required)
-- From: strings.lua
-- What: Tests various string.format flag combinations required by ISO C: # for octal/hex, 0 for zero-padding, - for left-align, + for sign, and precision specifiers.

do
assert(string.format("%#12o", 10) == "         012")
assert(string.format("%#10x", 100) == "      0x64")
assert(string.format("%#-17X", 100) == "0X64             ")
assert(string.format("%013i", -100) == "-000000000100")
assert(string.format("%2.5d", -100) == "-00100")
assert(string.format("%.u", 0) == "")
assert(string.format("%+#014.0f", 100) == "+000000000100.")
assert(string.format("%-16c", 97) == "a               ")
assert(string.format("%+.3G", 1.5) == "+1.5")
assert(string.format("%.0s", "alo")  == "")
assert(string.format("%.s", "alo")  == "")

assert(string.match(string.format("% 1.0E", 100), "^ 1E%+0+2$"))
assert(string.match(string.format("% .1g", 2^10), "^ 1e%+0+3$"))
end
