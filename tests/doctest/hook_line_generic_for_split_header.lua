-- A generic for whose explist or 'do' sits on another line than the 'for'
-- traces like reference: the nil padding for missing iterator values takes
-- the explist's last line, and the TFORCALL reached from TFORPREP and the
-- TFORLOOP after each TFORCALL are not traced (reference reaches both with
-- a goto that skips the hook).
local function trace(src)
  local f = load(src, "=c")
  local t = {}
  debug.sethook(function(e, l)
    if debug.getinfo(2, "S").source == "=c" then t[#t + 1] = l end
  end, "l")
  local ok, err = pcall(f)
  debug.sethook()
  print(table.concat(t, " "), ok or err)
end
trace("local t = {1, 2}\nfor k, v in next,\n t do\n local x = v\nend")
--> =1 2 3 4 2 4 2 5	true
trace("local t = {1, 2}\nfor k, v in pairs(t)\ndo\n local x = v\nend")
--> =1 2 3 4 2 4 2 5	true
trace("for k in\n nil\n do end")
--> =2 3	c:2: attempt to call a nil value (for iterator 'for iterator')
trace("for k, v in\n pairs({a=1})\n do\n local z = k\n end")
--> =2 3 4 2 5	true
trace("local t = {1, 2}\nfor k, v in ipairs(t) do local x = v end")
--> =1 2 2 2	true
trace("for k in next, {1,2,3} do\n if k == 2 then break end\nend")
--> =1 2 1 2 3	true
