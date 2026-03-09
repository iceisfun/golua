-- error_constant_annotation_short_circuit
--
-- Ensure `(constant '...')` annotations are only emitted when the runtime
-- operand actually comes from that constant. Short-circuit expressions can
-- skip the constant load and produce booleans at runtime, so those errors
-- must not include a constant annotation.
do
    -- Baseline: direct constant operands should keep the annotation.
    print(pcall(function() return ("hello")() end))
    --> ~^false\t.*attempt to call a string value \(constant 'hello'\)$

    print(pcall(function() return "10" | 1 end))
    --> ~^false\t.*attempt to perform bitwise operation on a string value \(constant '10'\)$

    -- Short-circuit false/true paths: runtime operand is boolean, not constant.
    print(pcall(function() return #(false and "b") end))
    --> ~^false\t.*attempt to get length of a boolean value$

    print(pcall(function() return #(true or "b") end))
    --> ~^false\t.*attempt to get length of a boolean value$

    print(pcall(function() return (false and "x") | 1 end))
    --> ~^false\t.*attempt to perform bitwise operation on a boolean value$

    print(pcall(function() return (false and "x")() end))
    --> ~^false\t.*attempt to call a boolean value$

    print(pcall(function() return (false and "x") .. 1 end))
    --> ~^false\t.*attempt to concatenate a boolean value$
end
