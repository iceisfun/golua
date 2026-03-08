-- Test: pack/unpack/packsize reject invalid alignment values

-- !0 should be rejected with "integral size (0) out of limits [1,16]"
local ok, err = pcall(string.packsize, "!0 i4")
assert(not ok, "!0 should be rejected")
assert(string.find(err, "integral size %(0%) out of limits"), "got: " .. tostring(err))

local ok2, err2 = pcall(string.pack, "!0 i4", 1)
assert(not ok2, "!0 should be rejected in pack")
assert(string.find(err2, "integral size %(0%) out of limits"), "got: " .. tostring(err2))

-- !3 should be rejected with "format asks for alignment not power of 2"
local ok3, err3 = pcall(string.packsize, "!3 i4")
assert(not ok3, "!3 should be rejected")
assert(string.find(err3, "not power of 2"), "got: " .. tostring(err3))

local ok4, err4 = pcall(string.pack, "!3 i4", 1)
assert(not ok4, "!3 should be rejected in pack")
assert(string.find(err4, "not power of 2"), "got: " .. tostring(err4))

-- Valid alignments should work
assert(pcall(string.packsize, "!1 i4"))
assert(pcall(string.packsize, "!2 i4"))
assert(pcall(string.packsize, "!4 i4"))
assert(pcall(string.packsize, "!8 i4"))
assert(pcall(string.packsize, "!16 i4"))

print("OK")
