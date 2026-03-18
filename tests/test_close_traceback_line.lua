-- When a <close> metamethod errors, the traceback frame above the close
-- should reference a line near the scope exit (end keyword), not the
-- block-opening keyword (do/for/while).

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

-- The 'do' is on line 6, 'end' is on line 10.
-- The frame line should be 10 (the end), not 6 (the do).
assert(frame_line and frame_line >= 10,
    string.format("__close frame should reference 'end' line (>=10), got line %s\n%s",
        tostring(frame_line), tostring(err)))
