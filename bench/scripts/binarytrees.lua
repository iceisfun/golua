local function make(depth)
    if depth == 0 then
        return {}
    end
    return { left = make(depth - 1), right = make(depth - 1) }
end

local function check(node)
    if node.left then
        return 1 + check(node.left) + check(node.right)
    end
    return 1
end

local root = make(16)
local count = check(root)
