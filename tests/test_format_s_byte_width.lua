-- Bug: string.format %s width/precision counts runes instead of bytes.
-- Lua 5.4 (like C sprintf) counts bytes for width padding and precision truncation.
-- GoLua delegates to Go's fmt.Sprintf which counts runes.

do
  -- 2-byte UTF-8 char: width 5 should pad based on byte length (2), adding 3 spaces
  assert(#string.format("%5s", "\xc3\xa9") == 5,
    "width should count bytes, got length: " .. #string.format("%5s", "\xc3\xa9"))

  -- 3-byte UTF-8 char: width 5 should pad based on byte length (3), adding 2 spaces
  assert(#string.format("%5s", "\xe2\x82\xac") == 5,
    "width should count bytes for 3-byte char, got length: " .. #string.format("%5s", "\xe2\x82\xac"))

  -- %.1s should truncate to 1 byte (not 1 rune)
  assert(#string.format("%.1s", "\xe2\x80\x99") == 1,
    "precision should truncate bytes, got length: " .. #string.format("%.1s", "\xe2\x80\x99"))

  -- %.4s on "café" (5 bytes) should truncate to 4 bytes
  assert(#string.format("%.4s", "caf\xc3\xa9") == 4,
    "precision should truncate to 4 bytes, got length: " .. #string.format("%.4s", "caf\xc3\xa9"))

  -- ASCII strings should be unaffected
  assert(string.format("%5s", "hi") == "   hi")
  assert(string.format("%.3s", "hello") == "hel")

end
