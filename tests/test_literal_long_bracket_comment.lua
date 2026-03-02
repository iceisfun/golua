-- Test: Long bracket comment
-- From: literals.lua
-- What: Tests that multi-line long bracket comments are correctly ignored by the parser, even when they contain syntax that would otherwise be invalid.

--[===[
x y z [==[ blu foo
]==
]
]=]==]
error error]=]===]
