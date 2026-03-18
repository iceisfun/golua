-- xpcall handler that triggers stack overflow should produce "error in error handling"
-- When pcall encounters a stack overflow while already handling one
-- inside an xpcall handler, it should produce "error in error handling"

do
    local function loop() return 1 + loop() end
    local _, msg = xpcall(loop, function(m)
        local ok, err = pcall(loop)
        return err
    end)
    print(msg)
    --> =error in error handling
end
