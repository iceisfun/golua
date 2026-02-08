-- goto.lua: Tests for goto/label scope validation per Lua 5.4 §8.2.15.

local function assert_eq(a, b, msg)
    if a ~= b then
        error((msg or "assertion failed") .. ": expected " .. tostring(b) .. ", got " .. tostring(a), 2)
    end
end

--------------------------------------------------------------------------------
-- Valid goto: forward jump
--------------------------------------------------------------------------------

goto skip_forward
error("should not execute")
::skip_forward::

--------------------------------------------------------------------------------
-- Valid goto: backward jump (loop-style)
--------------------------------------------------------------------------------

local counter = 0
::loop_start::
counter = counter + 1
if counter < 3 then
    goto loop_start
end
assert_eq(counter, 3, "backward goto loop")

--------------------------------------------------------------------------------
-- Valid goto: jump out of local scope
--------------------------------------------------------------------------------

do
    local y = 7
    goto out_of_scope
    error("should not execute")
end
::out_of_scope::

--------------------------------------------------------------------------------
-- Valid goto: jump over local declarations at same block level
-- (goto to label in same block, no new locals between goto and label)
--------------------------------------------------------------------------------

goto same_block
::same_block::

--------------------------------------------------------------------------------
-- Invalid: goto into scope of local variable (must be compile-time error)
-- load() returns nil + errmsg on compile errors
--------------------------------------------------------------------------------

local fn, errmsg = load([[
    goto bad
    local x = 42
    ::bad::
    print(x)
]])
assert(fn == nil, "goto into local scope must fail to compile, but got function")
assert(type(errmsg) == "string", "expected error message string from goto-into-scope")

--------------------------------------------------------------------------------
-- Invalid: goto to label in different (exited) block (must be compile-time error)
--------------------------------------------------------------------------------

local fn2, errmsg2 = load([[
    do
        ::inner::
    end
    goto inner
]])
assert(fn2 == nil, "goto to label in exited block must fail to compile, but got function")
assert(type(errmsg2) == "string", "expected error message string from label-visibility")

--------------------------------------------------------------------------------
-- Valid: goto within nested blocks to label in outer block
--------------------------------------------------------------------------------

::outer_label::
do
    do
        if counter > 100 then
            goto outer_label
        end
    end
end

--------------------------------------------------------------------------------
-- Valid: label and goto in same block, label before goto (backward)
--------------------------------------------------------------------------------

local j = 0
::backward_label::
j = j + 1
if j < 2 then
    goto backward_label
end
assert_eq(j, 2, "backward goto with label in same block")
