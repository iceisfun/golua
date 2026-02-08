-- BROKEN: utf8 standard library not yet implemented
-- Tracking: Phase 2 (new subsystems)
--
-- The utf8.* library (utf8.char, utf8.codes, utf8.codepoint, utf8.len, utf8.offset)
-- is not yet available. These tests document expected behavior for future implementation.

assert(utf8 ~= nil, "utf8 library should exist")
assert(utf8.len("hello") == 5)
