-- ============================================================================
-- Test: Variadic Argument Propagation and Selection
--
-- PURPOSE:
-- This test validates that the VM correctly manages the '...' (vararg)
-- expression across multiple function boundaries and properly integrates
-- with the 'select' built-in.
-- ============================================================================

-- f(...) serves as a consumer of variadic arguments.
-- It utilizes 'select(n, ...)' to verify that the VM correctly indexes
-- into the vararg list passed from the caller.
local function f(...)
    -- select(3, ...) returns the 3rd element and all subsequent elements.
    -- In this specific test, we expect it to return 3.
    return select(3, ...)
end

-- g(...) acts as a passthrough proxy.
-- It tests whether '...' can be forwarded into another function call
-- without losing its identity or causing stack corruption.
local function g(...)
    return f(...)
end

-- ASSERTION: Ensure that the 3rd positional argument (3) is successfully
-- recovered after traveling through two layers of variadic passing.
assert(g(1, 2, 3, 4) == 3)


-- ============================================================================
-- Test: Tail-Call Vararg Integrity
--
-- PURPOSE:
-- Verifies that the VM's Tail Call Optimization (TCO) correctly preserves
-- the full list of return values when a function returns the result of
-- another variadic function call.
-- ============================================================================

-- passthrough(...) returns the entire variadic list back to the caller.
local function passthrough(...)
    return ...
end

-- tail(...) executes a tail call.
-- The VM should not create a new stack frame for passthrough(...),
-- but it must ensure the return values (1, 2, 3) are correctly placed
-- onto the calling frame's registers.
local function tail(...)
    return passthrough(...)
end

-- ASSERTION: Multi-assignment capture.
-- Validates that the vararg list (1, 2, 3) survived the tail-call
-- optimization and reached the local assignment scope intact.
local a, b, c = tail(1, 2, 3)
assert(a == 1 and b == 2 and c == 3)
