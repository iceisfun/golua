-- Lua 5.5 global declaration compile-time checking

-- Basic: declared global can be read and written
do
  global X
  global print
  X = 1
  print(X)
  --> =1
end

-- Undeclared read: error when explicit globals are active
do
  local ok, err = load("global X; global print; print(Y)")
  print(ok == nil)
  --> =true
  print(err:find("variable 'Y' not declared") ~= nil)
  --> =true
end

-- Undeclared write: error when explicit globals are active
do
  local ok, err = load("global X; Y = 1")
  print(ok == nil)
  --> =true
  print(err:find("variable 'Y' not declared") ~= nil)
  --> =true
end

-- Const write error
do
  local ok, err = load("global<const> X; X = 1")
  print(ok == nil)
  --> =true
  print(err:find("attempt to assign to const variable 'X'") ~= nil)
  --> =true
end

-- Const read OK
do
  global<const> print
  print("hello")
  --> =hello
end

-- Wildcard: global * allows all names
do
  global print
  global *
  Y = 1
  print(Y)
  --> =1
end

-- Const wildcard: read OK, write error
do
  local ok, err = load("global<const> *; X = 1")
  print(ok == nil)
  --> =true
  print(err:find("attempt to assign to const variable 'X'") ~= nil)
  --> =true
end

-- Const wildcard read OK
do
  global<const> *
  print("hi")
  --> =hi
end

-- Specific name overrides const wildcard
do
  global print
  global<const> *
  print = print -- assigning to explicitly declared rw name is OK
  print("ok")
  --> =ok
end

-- Nested scopes inherit
do
  global X
  global print
  do
    X = 1
    print(X)
    --> =1
  end
end

-- Nested scope with own decls
do
  global X
  global print
  do
    global Y
    Y = 1
    X = 2
    print(X + Y)
    --> =3
  end
end

-- Inner scope explicit global voids implicit for inner scope only
do
  local ok, err = load([[
    do
      global Y
      X = 1
    end
  ]])
  print(ok == nil)
  --> =true
  print(err:find("variable 'X' not declared") ~= nil)
  --> =true
end

-- Function body inherits outer global restrictions
do
  local ok, err = load("global X; local f = function() Y = 1 end")
  print(ok == nil)
  --> =true
  print(err:find("variable 'Y' not declared") ~= nil)
  --> =true
end

-- Function body can use names declared in outer scope
do
  global X
  global print
  local f = function() X = 42 end
  f()
  print(X)
  --> =42
end

-- Function body can add own global declarations
do
  local ok, err = load("global X; local f = function() global Y; Y = 1 end")
  print(ok ~= nil)
  --> =true
end

-- global function declaration
do
  global function f() return 42 end
  global print
  print(f())
  --> =42
end

-- Cumulative: multiple global statements in same scope
do
  global X
  global Y
  global print
  X = 1
  Y = 2
  print(X + Y)
  --> =3
end

-- Scope exit restores: after leaving inner scope, outer rules apply
do
  global print
  global *
  do
    global X
    -- only X is declared in this inner scope
  end
  -- back to outer scope: global * is active
  Z = 99
  print(Z)
  --> =99
end

-- global function also registers name for subsequent use
do
  global function myFunc() return "hi" end
  global print
  print(myFunc())
  --> =hi
end
