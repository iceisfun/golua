-- When a <close> metamethod errors, the traceback frame above the close
-- should reference the line of the block's LAST STATEMENT (reference Lua's
-- ls->lastline at leaveblock), not the block-opening keyword (do) nor the
-- closing 'end'. Verified against lua5.5.0.

local ok, err = xpcall(function()
    do
        local x <close> = setmetatable({}, {
            __close = function() error("boom") end
        })
    end
end, debug.traceback)

-- Find the enclosing frame: the line AFTER "in metamethod 'close'"
local found_close = false
local frame_line = nil
for line in tostring(err):gmatch("[^\n]+") do
    if found_close and not frame_line then
        frame_line = tonumber(line:match(":(%d+):"))
    end
    if line:find("metamethod 'close'") then
        found_close = true
    end
end

-- The <close> declaration is the block's last statement and spans lines 8-10;
-- its final token "})" is on line 10. Reference Lua attributes the block-exit
-- close (and thus the enclosing frame) to that line, NOT the 'do' (line 7) or
-- the 'end' (line 11).
assert(frame_line == 10,
    string.format("__close frame should reference the last-statement line (10), got line %s\n%s",
        tostring(frame_line), tostring(err)))
