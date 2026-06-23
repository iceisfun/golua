-- string.gsub with a TABLE replacement keys off only the first capture
-- (Lua's push_onecapture(0)), so only that capture must be finished. A pattern
-- like "()(" has a finished position capture #1 and an unfinished capture #2;
-- the table path never touches #2, so it must succeed (unlike the function
-- path, which materializes every capture and errors on the unfinished one).
-- Matches reference Lua 5.5.

-- finished first capture, unfinished second -> table path succeeds
print(pcall(string.gsub, "ab", "()(", {}))
--> =true	ab	3

-- the first capture itself unfinished -> "unfinished capture"
print(pcall(string.gsub, "ab", "(", {}))
--> =false	unfinished capture

-- position capture #1 used as the table key
print(pcall(string.gsub, "xy", "()", {[1] = "A"}))
--> =true	Axy	3

-- the function path DOES materialize every capture, so it still errors
print(pcall(string.gsub, "ab", "()(", function() return "z" end))
--> =false	unfinished capture
