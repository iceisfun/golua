-- test_string_find: string.find pattern and plain mode

-- Pattern matching with captures
do
    local text = "abc123def456"

    local start_pos, end_pos = string.find(text, "%d+")
    assert(start_pos == 4 and end_pos == 6, string.format("expected match 4..6, got %s..%s", tostring(start_pos), tostring(end_pos)))

    local s2, e2, digits = string.find(text, "(%d+)")
    assert(digits == "123", string.format("expected capture '123', got %s", tostring(digits)))
    assert(s2 == 4 and e2 == 6, string.format("expected capture bounds 4..6, got %s..%s", tostring(s2), tostring(e2)))

    -- When plain flag is set, the pattern should be treated as a literal
    local plain_start = select(1, string.find(text, "%d+", 1, true))
    assert(plain_start == nil, "plain search should treat % as literal and fail to match")
end

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
