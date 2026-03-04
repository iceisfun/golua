-- Test: Invalid format errors
-- From: tpack.lua
-- What: Tests error messages for invalid pack format options: out-of-limits sizes, invalid characters, non-power-of-2 alignment, missing sizes, and variable-length format in packsize.

do
local pack = string.pack
local packsize = string.packsize
local unpack = string.unpack
local NB = 16

local function checkerror (msg, f, ...)
  local status, err = pcall(f, ...)
  assert(not status and string.find(err, msg))
end

checkerror("out of limits", pack, "i0", 0)
checkerror("out of limits", pack, "i" .. NB + 1, 0)
checkerror("out of limits", pack, "!" .. NB + 1, 0)
checkerror("%(17%) out of limits %[1,16%]", pack, "Xi" .. NB + 1)
checkerror("invalid format option 'r'", pack, "i3r", 0)
checkerror("16%-byte integer", unpack, "i16", string.rep('\3', 16))
checkerror("not power of 2", pack, "!4i3", 0);
checkerror("missing size", pack, "c", "")
checkerror("variable%-length format", packsize, "s")
checkerror("variable%-length format", packsize, "z")
end
