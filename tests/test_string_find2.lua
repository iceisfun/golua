-- test_string_find2: string.find plain and pattern mode edge cases

-- Plain mode
do
    local s, e = string.find("a", "a", 1, true)
    assert(s == 1 and e == 1, "find plain single char")
end

do
    local s, e = string.find("hello world!", "o w", 1, true)
    assert(s == 5 and e == 7, "find plain multi-char")
end

do
    assert(string.find("xyzt", "yt", 1, true) == nil, "find plain no match")
end

-- Pattern mode
do
    local s, e = string.find("1234 abc453", "%l+")
    assert(s == 6 and e == 8, "find pattern %l+")
end

-- Pattern with captures
do
    local s, e, c1, c2 = string.find("  foo=[a [lovely] day];", "(%w+)=(%b[])")
    assert(s == 3 and e == 22, "find captures positions")
    assert(c1 == "foo", "find capture 1")
    assert(c2 == "[a [lovely] day]", "find capture 2")
end

-- Start position past string
do
    assert(string.find("x", "x", 3) == nil, "find past end")
end

-- Negative start position
do
    local s, e = string.find("x", "x", -10)
    assert(s == 1 and e == 1, "find negative start")
end

-- No match
do
    assert(string.find("x", "y") == nil, "find no match")
end
