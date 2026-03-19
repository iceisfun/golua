-- After pcall catches an error, subsequent uncaught errors should not
-- include stale frames from the pcall in their traceback.

pcall(error, "first")

-- The traceback for this error should show only the current stack,
-- not leaked frames from the pcall above.
local ok, err = pcall(function()
    error("second")
end)
-- The error message should reference the error() call site, not pcall
print(err:find("second", 1, true) ~= nil)
--> =true
