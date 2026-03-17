-- debug.getinfo with invalid option '>' alone should include
-- the offending character in the error message.

-- '>' alone is not valid (it must follow another option letter)
local ok1, err1 = pcall(debug.getinfo, 1, ">")
print(err1:find("'>'") ~= nil)
--> =true

-- Valid options followed by invalid don't show the char (lua5.4 quirk)
-- Just verify it errors
local ok2, err2 = pcall(debug.getinfo, 1, "Sx")
print(ok2)
--> =false
