-- Function-form debug.getlocal() should return nil for stripped bytecode when
-- local names are unavailable.

local f = load(string.dump(function(a)
    return a
end, true))

print(debug.getlocal(f, 1))
--> =nil
