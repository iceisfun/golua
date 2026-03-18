-- When a <close> metamethod errors, the traceback should reference the
-- line of the 'end' keyword (where the close executes), not the line
-- of the block-opening keyword.

local ok, err = pcall(function()
    do                                                             -- line 6
        local x <close> = setmetatable({}, {                      -- line 7
            __close = function() error("boom") end
        })
    end                                                            -- line 10
end)

-- The error should reference line 10 (the 'end'), not line 6 (the 'do')
-- Extract the line number from the error traceback
local line = tostring(err):match(":(%d+): boom")
print(line)
--> =10
