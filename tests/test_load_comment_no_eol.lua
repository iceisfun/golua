-- Test: load with comment without EOL
-- From: strings.lua
-- What: Tests that load can handle a chunk ending with a comment that has no trailing newline.

do
assert(load("return 1\n--comment without ending EOL")() == 1)
end
