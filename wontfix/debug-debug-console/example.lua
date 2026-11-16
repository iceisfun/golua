-- debug.debug is reference Lua's interactive console: it reads lines from
-- the C stdin and runs each one until a line reading "cont". golua does not
-- provide it.
print(type(debug.debug))
-- golua:     nil
-- reference: function
