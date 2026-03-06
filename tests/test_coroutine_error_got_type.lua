-- Coroutine functions include "got TYPE" in error messages

-- coroutine.create
local ok, err = pcall(coroutine.create, 42)
assert(err:find("got number"), "create(42): " .. err)
local ok, err = pcall(coroutine.create)
assert(err:find("got no value"), "create(): " .. err)

-- coroutine.wrap
local ok, err = pcall(coroutine.wrap, "x")
assert(err:find("got string"), "wrap('x'): " .. err)
local ok, err = pcall(coroutine.wrap)
assert(err:find("got no value"), "wrap(): " .. err)

-- coroutine.status
local ok, err = pcall(coroutine.status, true)
assert(err:find("got boolean"), "status(true): " .. err)
local ok, err = pcall(coroutine.status)
assert(err:find("got no value"), "status(): " .. err)

-- coroutine.close
local ok, err = pcall(coroutine.close, 42)
assert(err:find("got number"), "close(42): " .. err)
local ok, err = pcall(coroutine.close)
assert(err:find("got no value"), "close(): " .. err)

-- coroutine.resume
local ok, err = pcall(coroutine.resume, "x")
assert(err:find("got string"), "resume('x'): " .. err)

print("PASS")
