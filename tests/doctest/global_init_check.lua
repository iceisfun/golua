-- global init runtime check: error if variable already has a non-nil value

-- Simple global init works
do
    global X = 10
    global print
    print(X)
    --> =10
end

-- global declaration (no init) when variable exists: OK
do
    Y = 42
    global Y, print
    print(Y)
    --> =42
end

-- global declaration then assign: OK
do
    global Z, print
    Z = 99
    print(Z)
    --> =99
end

-- Error: global init when already defined via plain assignment
do
    A = 5
    local ok, err = pcall(load("global A = 10"))
    print(ok)
    --> =false
    print(tostring(err):find("global 'A' already defined") ~= nil)
    --> =true
end

-- Error: double global init
do
    global B = 1
    global pcall, load, print, tostring
    local ok, err = pcall(load("global B = 2"))
    print(ok)
    --> =false
    print(tostring(err):find("global 'B' already defined") ~= nil)
    --> =true
end

-- Error: global function when already defined
do
    global C = 100
    global pcall, load, print, tostring
    local ok, err = pcall(load("global function C() end"))
    print(ok)
    --> =false
    print(tostring(err):find("global 'C' already defined") ~= nil)
    --> =true
end

-- pcall catches the error cleanly
do
    global D = "hello"
    global pcall, load, print, type
    local ok, err = pcall(load("global D = 'world'"))
    print(ok)
    --> =false
    print(type(err))
    --> =string
end
