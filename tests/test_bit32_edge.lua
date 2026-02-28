-- test_bit32_edge: bit32 edge cases (arshift, shifts, truncation, extract/replace, rotate)

-- arshift
do
    -- sign extension basic
    local r = bit32.arshift(0x80000000, 1)
    assert(r == 0xC0000000, "arshift sign extension basic: expected 0xC0000000, got " .. string.format("0x%X", r))

    -- sign extension fills all bits
    r = bit32.arshift(0x80000000, 31)
    assert(r == 0xFFFFFFFF, "arshift fills all bits")

    -- sign extension shift >= 32
    r = bit32.arshift(0x80000000, 32)
    assert(r == 0xFFFFFFFF, "arshift >= 32 with sign bit")

    -- no sign extension when MSB clear
    r = bit32.arshift(0x40000000, 1)
    assert(r == 0x20000000, "arshift no sign extension: expected 0x20000000, got " .. string.format("0x%X", r))

    -- no sign extension shift >= 32
    r = bit32.arshift(0x7FFFFFFF, 32)
    assert(r == 0, "arshift >= 32 with clear sign bit")

    -- arshift by 0
    assert(bit32.arshift(0xDEADBEEF, 0) == 0xDEADBEEF, "arshift by 0")

    -- arshift negative disp is lshift
    assert(bit32.arshift(1, -4) == 16, "arshift negative disp is lshift")

    -- arshift 0xFF000000 by 8
    r = bit32.arshift(0xFF000000, 8)
    assert(r == 0xFFFF0000, "arshift 0xFF000000 by 8: expected 0xFFFF0000, got " .. string.format("0x%X", r))

    -- all ones stays all ones
    assert(bit32.arshift(0xFFFFFFFF, 1) == 0xFFFFFFFF, "arshift all ones by 1")
    assert(bit32.arshift(0xFFFFFFFF, 16) == 0xFFFFFFFF, "arshift all ones by 16")
    assert(bit32.arshift(0xFFFFFFFF, 31) == 0xFFFFFFFF, "arshift all ones by 31")
end

-- Shift edge cases
do
    assert(bit32.lshift(0xFFFFFFFF, 32) == 0, "lshift by 32 returns 0")
    assert(bit32.rshift(0xFFFFFFFF, 32) == 0, "rshift by 32 returns 0")
    assert(bit32.lshift(1, 100) == 0, "lshift large displacement")
    assert(bit32.rshift(1, 100) == 0, "rshift large displacement")
    assert(bit32.lshift(0x100, -4) == 0x10, "lshift negative is rshift")
    assert(bit32.rshift(0x10, -4) == 0x100, "rshift negative is lshift")

    local r = bit32.rshift(0x80000000, 1)
    assert(r == 0x40000000, "rshift does not sign extend: expected 0x40000000, got " .. string.format("0x%X", r))
end

-- Input truncation
do
    assert(bit32.band(0x1FFFFFFFF, 0xFFFFFFFF) == 0xFFFFFFFF, "large input truncated to uint32")
    assert(bit32.band(-1, 0xFF) == 0xFF, "negative input wraps as uint32")
    assert(bit32.bnot(-1) == 0, "bnot(-1) should be 0 since -1 truncates to 0xFFFFFFFF")
end

-- Extract and replace
do
    assert(bit32.extract(0xAC, 4, 4) == 10, "extract nibble")
    assert(bit32.extract(5, 0) == 1, "extract bit 0 of 5")
    assert(bit32.extract(5, 1) == 0, "extract bit 1 of 5")
    assert(bit32.extract(5, 2) == 1, "extract bit 2 of 5")
    assert(bit32.extract(0xDEADBEEF, 0, 32) == 0xDEADBEEF, "extract full 32 bits")
    assert(bit32.replace(0xF0, 0x0A, 0, 4) == 0xFA, "replace low nibble")
    assert(bit32.replace(0, 1, 7) == 128, "replace single bit")

    -- Extract/replace roundtrip
    local orig = 0xDEADBEEF
    local hi = bit32.extract(orig, 16, 16)
    local lo = bit32.extract(orig, 0, 16)
    local rebuilt = bit32.replace(0, lo, 0, 16)
    rebuilt = bit32.replace(rebuilt, hi, 16, 16)
    assert(rebuilt == orig, "extract/replace roundtrip failed")
end

-- Rotate
do
    assert(bit32.lrotate(0x80000000, 1) == 1, "lrotate wraps MSB to LSB")
    assert(bit32.rrotate(1, 1) == 0x80000000, "rrotate wraps LSB to MSB")
    assert(bit32.lrotate(0x12345678, 32) == 0x12345678, "full lrotation is identity")
    assert(bit32.rrotate(0x12345678, 32) == 0x12345678, "full rrotation is identity")
    assert(bit32.lrotate(1, -1) == 0x80000000, "negative lrotate is rrotate")
    assert(bit32.rrotate(2, -1) == 4, "negative rrotate is lrotate")
end
