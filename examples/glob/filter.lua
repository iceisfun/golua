-- filter.lua: A small utility module that uses glob for filtering lists.
--
-- Expects the "glob" table to be available in the global environment
-- (provided by the Go host).
--
-- Usage from Lua:
--   local filter = dofile("filter.lua")
--   local matches = filter.select(items, "alpha-*")
--   local first   = filter.first(items, "*-100")
--   local idx     = filter.find(items, "beta-*")

local M = {}

-- select returns all items matching the pattern.
function M.select(items, pattern)
    local result = {}
    for _, item in ipairs(items) do
        if glob.match(pattern, item) then
            result[#result + 1] = item
        end
    end
    return result
end

-- first returns the first item matching the pattern, or nil.
function M.first(items, pattern)
    for _, item in ipairs(items) do
        if glob.match(pattern, item) then
            return item
        end
    end
    return nil
end

-- find returns the index of the first matching item, or nil.
function M.find(items, pattern)
    for i, item in ipairs(items) do
        if glob.match(pattern, item) then
            return i
        end
    end
    return nil
end

-- reject returns all items that do NOT match the pattern.
function M.reject(items, pattern)
    local result = {}
    for _, item in ipairs(items) do
        if not glob.match(pattern, item) then
            result[#result + 1] = item
        end
    end
    return result
end

return M
