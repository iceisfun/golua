-- enemy.lua: Lua companion module.
--
-- Wraps the Go-provided enemy table with a metatable to add
-- higher-level behavior: AI decisions, policy, and composition.
-- Go owns state; Lua owns behavior.

local Enemy = {}
Enemy.__index = Enemy

-- Wrap a Go-provided enemy table into a Lua Enemy object.
-- The Go table is stored under _core (private by convention).
function Enemy.wrap(go_enemy)
    local self = setmetatable({}, Enemy)
    self._core = go_enemy
    return self
end

-- Accessor: get the enemy's name.
function Enemy:name()
    return self._core.name()
end

-- Accessor: get current health.
function Enemy:health()
    return self._core.health()
end

-- Accessor: get maximum health.
function Enemy:max_hp()
    return self._core.max_hp()
end

-- Accessor: get position as x, y.
function Enemy:position()
    return self._core.position()
end

-- Accessor: check if alive.
function Enemy:is_alive()
    return self._core.is_alive()
end

-- Accessor: health as a fraction (0.0 to 1.0).
function Enemy:health_pct()
    return self:health() / self:max_hp()
end

-- Policy: describe current status as a string.
function Enemy:status()
    if not self:is_alive() then
        return self:name() .. " is dead"
    end
    local x, y = self:position()
    return string.format("%s: %d/%d HP at (%.1f, %.1f)",
        self:name(), self:health(), self:max_hp(), x, y)
end

-- Policy: decide whether to flee (below 25% health).
function Enemy:should_flee()
    if not self:is_alive() then
        return false
    end
    return self:health() < self:max_hp() * 0.25
end

-- Policy: decide whether to heal (below 50% health).
function Enemy:should_heal()
    if not self:is_alive() then
        return false
    end
    return self:health() < self:max_hp() * 0.5
end

-- Behavior: take a turn with simple AI.
-- Returns a string describing the action taken.
function Enemy:take_turn(target_x, target_y)
    if not self:is_alive() then
        return self:name() .. " is dead, skipping turn"
    end

    -- Priority 1: flee if critically wounded
    if self:should_flee() then
        -- Move away from target
        local x, y = self:position()
        local dx = x - target_x
        local dy = y - target_y
        -- Normalize and move 5 units away
        local dist = math.sqrt(dx * dx + dy * dy)
        if dist > 0 then
            self._core.move_to(x + (dx / dist) * 5, y + (dy / dist) * 5)
        end
        return self:name() .. " flees!"
    end

    -- Priority 2: heal if wounded
    if self:should_heal() then
        self._core.heal(10)
        return self:name() .. " heals for 10"
    end

    -- Priority 3: advance toward target
    local x, y = self:position()
    local dx = target_x - x
    local dy = target_y - y
    local dist = math.sqrt(dx * dx + dy * dy)
    if dist > 2 then
        -- Move 2 units toward target
        self._core.move_to(x + (dx / dist) * 2, y + (dy / dist) * 2)
        return self:name() .. " advances"
    end

    return self:name() .. " holds position"
end

return Enemy
