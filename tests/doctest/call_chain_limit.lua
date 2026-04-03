-- __call metamethod chain depth limit (Lua 5.5: max 15)

-- 1 level of __call: OK
do
    local t = setmetatable({}, {__call = function(self) return "ok1" end})
    print(t())
    --> =ok1
end

-- 14 levels: OK
do
    local chain = {}
    for i = 1, 14 do
        local prev = chain[#chain]
        local t = setmetatable({}, {__call = prev or function() return "ok14" end})
        chain[#chain+1] = t
    end
    print(chain[14]())
    --> =ok14
end

-- 15 levels: OK (the limit allows 15 __call resolutions)
do
    local chain = {}
    for i = 1, 15 do
        local prev = chain[#chain]
        local t = setmetatable({}, {__call = prev or function() return "ok15" end})
        chain[#chain+1] = t
    end
    print(chain[15]())
    --> =ok15
end

-- 16 levels: error
do
    local chain = {}
    for i = 1, 16 do
        local prev = chain[#chain]
        local t = setmetatable({}, {__call = prev or function() return "deep" end})
        chain[#chain+1] = t
    end
    local ok, err = pcall(chain[16])
    print(ok)
    --> =false
    print(tostring(err):find("too long") ~= nil)
    --> =true
end

-- Self-referencing __call (infinite loop): error
do
    local t = {}
    setmetatable(t, {__call = t})
    local ok, err = pcall(t)
    print(ok)
    --> =false
    print(tostring(err):find("too long") ~= nil)
    --> =true
end

-- Tail-call position: same limit applies
do
    local chain = {}
    for i = 1, 16 do
        local prev = chain[#chain]
        local t = setmetatable({}, {__call = prev or function() return "deep" end})
        chain[#chain+1] = t
    end
    local ok, err = pcall(function() return chain[16]() end)
    print(ok)
    --> =false
    print(tostring(err):find("too long") ~= nil)
    --> =true
end
