-- utf8.codes should skip leading continuation bytes and only error on
-- malformed UTF-8 when attempting to decode a rune-start byte.

do
  local s = string.char(0xB2, 0xBD, 0xB8, 0x2A, 0x09, 0x5E)
  local out = {}
  for p, c in utf8.codes(s) do
    out[#out + 1] = p .. ":" .. c
  end
  assert(table.concat(out, ",") == "4:42,5:9,6:94")
end

do
  local s = string.char(0x94, 0x7E, 0x69, 0xE5)
  local ok, err = pcall(function()
    for _p, _c in utf8.codes(s) do
    end
  end)
  assert(ok == false)
  assert(string.find(tostring(err), "invalid UTF-8 code", 1, true) ~= nil)
end

do
  local s = string.char(0x61, 0xBA)
  local out = {}
  for p, c in utf8.codes(s) do
    out[#out + 1] = p .. ":" .. c
  end
  assert(table.concat(out, ",") == "1:97")
end
