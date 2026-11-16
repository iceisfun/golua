-- A to-be-closed variable that gets a non-closable value is named in the
-- error. When the function is stripped there is no local name, and reference
-- luaG_findlocal names the live register "(temporary)", not "?".

local function stripped(src)
    return load(string.dump(load(src, "=chunk"), true))
end

print(pcall(stripped("local x <close> = 1")))
--> =false	?:?: variable '(temporary)' got a non-closable value

print(pcall(stripped("local a = 1 local x <close> = 1")))
--> =false	?:?: variable '(temporary)' got a non-closable value

print(pcall(stripped("for k in next, {}, nil, 1 do end")))
--> =false	?:?: variable '(temporary)' got a non-closable value

-- With debug information the real names are kept.
print(pcall(load("local x <close> = 1", "=chunk")))
--> =false	chunk:1: variable 'x' got a non-closable value

print(pcall(load("for k in next, {}, nil, 1 do end", "=chunk")))
--> =false	chunk:1: variable '(for state)' got a non-closable value
