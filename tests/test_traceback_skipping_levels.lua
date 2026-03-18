-- When a traceback is truncated due to depth, the middle section
-- should show "...\t(skipping N levels)" with the count of omitted
-- frames, matching Lua 5.4 behavior.

local function deep(n)
    if n == 0 then return debug.traceback("", 1) end
    local r = deep(n - 1); return r
end

local tb = deep(30)

-- Verify truncation happened
assert(tb:find("%.%.%."), "traceback should truncate at depth 30")

-- Verify the "(skipping N levels)" text is present
-- Look for it as a separate check to isolate the exact difference
local skip_line = nil
for line in tb:gmatch("[^\n]+") do
    if line:find("%.%.%.") then skip_line = line end
end
assert(skip_line:find("skipping %d+ levels"),
    "truncation line should say '(skipping N levels)', got: [" .. skip_line .. "]")
