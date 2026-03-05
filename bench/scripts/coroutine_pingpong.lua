local function worker()
    for i = 1, 100000 do
        coroutine.yield()
    end
end

local co = coroutine.create(worker)

while coroutine.status(co) ~= "dead" do
    coroutine.resume(co)
end
