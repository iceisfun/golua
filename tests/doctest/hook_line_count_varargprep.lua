-- With line and count hooks both set, a count event on a vararg function's
-- VARARGPREP must not hide the line event for its first line: VARARGPREP
-- carries that line, and like reference (L->oldpc = 1 after VARARGPREP) the
-- next instruction is still a new line. Main and load()ed chunks are vararg.
local f = load("local x = 0\nlocal y = 1\nlocal z = 2", "=chunk")
local out = {}
for n = 1, 6 do
  local t = {}
  debug.sethook(function(e, l)
    if e == "line" and debug.getinfo(2, "S").source == "=chunk" then
      t[#t + 1] = l
    end
  end, "l", n)
  f()
  debug.sethook()
  out[#out + 1] = table.concat(t, " ")
end
print(table.concat(out, ", "))
--> =1 2 3 3, 1 2 3, 1 2 3, 1 2 3, 1 2 3, 1 2 3
