-- debug.getinfo(1, "n") inside a tail-called function should return
-- name=nil, namewhat="" because the original caller frame is gone.

local function inner()
  local info = debug.getinfo(1, "n")
  -- In a tail call, the name information is lost
  assert(info.name == nil, "expected name=nil, got name=" .. tostring(info.name))
  assert(info.namewhat == "", "expected namewhat='', got namewhat=" .. tostring(info.namewhat))
end

local function outer()
  return inner()  -- tail call
end

outer()

-- Also verify that non-tail-call still gets name info
local function caller()
  inner()  -- regular call (not tail)
end

-- Redefine inner to check the non-tail case
local function inner2()
  local info = debug.getinfo(1, "n")
  assert(info.name == "inner2", "expected name='inner2', got name=" .. tostring(info.name))
  assert(info.namewhat == "upvalue", "expected namewhat='upvalue', got namewhat=" .. tostring(info.namewhat))
end

local function caller2()
  inner2()  -- regular call
  return nil
end

caller2()

-- Verify the tail call flag is set
local function check_tailcall()
  local info = debug.getinfo(1, "t")
  return info.istailcall
end

local function do_tailcall()
  return check_tailcall()  -- tail call
end

assert(do_tailcall() == true, "expected istailcall=true")

print("PASS")
