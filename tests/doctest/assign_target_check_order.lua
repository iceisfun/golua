-- Reference restassign checks each target of an assignment as it is parsed,
-- left to right and before the value list: a const target, or a global not
-- declared under a 'global' declaration, is reported in source order. The
-- stores themselves run right to left, but that must not change which error
-- a multiple assignment reports.

local function check(src)
    print((select(2, load(src, "=c"))))
end

check("global x <const>; x, y = 1")
--> =c:1: attempt to assign to const variable 'x'

check("global x <const>, t; x, t[z] = 1")
--> =c:1: attempt to assign to const variable 'x'

check("global x <const>, t; t[z], x = 1")
--> =c:1: variable 'z' not declared

check("local a <const>, b <const> = 1, 2; a, b = 3, 4")
--> =c:1: attempt to assign to const variable 'a'

check("local a <const> = {}; local b <const> = 2; a, b = 3, 4")
--> =c:1: attempt to assign to const variable 'a'

check("local a <const> = {}; local function f() local b <close> = nil; a, b = 1, 2 end")
--> =c:1: attempt to assign to const variable 'a'

check("global y, t; local a <const> = 1; t[1], y, a = 1")
--> =c:1: attempt to assign to const variable 'a'

-- The targets are checked before the value list.
check("global x; local a <const> = 1; a, x = q")
--> =c:1: attempt to assign to const variable 'a'

check("global none; local c <const> = 1; c, d = 1")
--> =c:1: attempt to assign to const variable 'c'

check("global *; global z <const>; x, z, w = 1")
--> =c:1: attempt to assign to const variable 'z'

check("global x, y; x, y = 1, z")
--> =c:1: variable 'z' not declared
