-- os.time{... isdst = true} under a time zone that has no daylight saving.
--
-- Run this with TZ=UTC (a zone with no DST):
--     TZ=UTC go run ./cmd/lua wontfix/os-time-isdst-no-dst-zone/example.lua
--     TZ=UTC lua5.5.0           wontfix/os-time-isdst-no-dst-zone/example.lua
--
-- Reference Lua delegates to C mktime(), which under glibc honors tm_isdst=1
-- even in a no-DST zone by subtracting 3600s (a glibc-specific behavior).
-- golua reports that the requested local time cannot be represented, because
-- no offset in the active zone corresponds to "DST is in effect".

local ok, res = pcall(function()
  return os.time{year = 2000, month = 6, day = 1, hour = 12, isdst = true}
end)
print(ok, res)
--> golua (TZ=UTC):    false   <...>: time result cannot be represented in this installation
--> lua5.5.0 (TZ=UTC): true    959857200

-- Under a real DST zone (e.g. TZ=America/New_York) golua and the reference
-- agree exactly, in both winter and summer. The divergence is specific to
-- asking for isdst=true in a zone that never observes DST.
