-- Test: UTF-8 escape sequences
-- From: literals.lua
-- What: Tests \u{XXXX} UTF-8 escape sequences in strings, including boundary values for 1-byte through 6-byte sequences.

do
  assert("\u{0}\u{00000000}\x00\0" == string.char(0, 0, 0, 0))

  -- limits for 1-byte sequences
  assert("\u{0}\u{7F}" == "\x00\x7F")

  -- limits for 2-byte sequences
  assert("\u{80}\u{7FF}" == "\xC2\x80\xDF\xBF")

  -- limits for 3-byte sequences
  assert("\u{800}\u{FFFF}" ==   "\xE0\xA0\x80\xEF\xBF\xBF")

  -- limits for 4-byte sequences
  assert("\u{10000}\u{1FFFFF}" == "\xF0\x90\x80\x80\xF7\xBF\xBF\xBF")

  -- limits for 5-byte sequences
  assert("\u{200000}\u{3FFFFFF}" == "\xF8\x88\x80\x80\x80\xFB\xBF\xBF\xBF\xBF")

  -- limits for 6-byte sequences
  assert("\u{4000000}\u{7FFFFFFF}" ==
         "\xFC\x84\x80\x80\x80\x80\xFD\xBF\xBF\xBF\xBF\xBF")
end
