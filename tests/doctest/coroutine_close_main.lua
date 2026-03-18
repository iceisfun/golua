-- coroutine.close on main thread from inside a coroutine
local main = coroutine.running()
;(coroutine.wrap(function ()
  local st, msg = pcall(coroutine.close, main)
  print(st, msg)
end))()
--> false	cannot close a normal coroutine
