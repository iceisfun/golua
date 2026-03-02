-- Test: pm.lua - string.gsub with function replacements
-- From: pm.lua
-- What: Tests string.gsub with function replacements including string.upper, rawset for globals, nested gsub, load/dostring, and position tracking.

do
  assert(string.gsub("um (dois) tres (quatro)", "(%(%w+%))", string.upper) ==
              "um (DOIS) tres (QUATRO)")

  do
    local function setglobal (n,v) rawset(_G, n, v) end
    string.gsub("a=roberto,roberto=a", "(%w+)=(%w%w*)", setglobal)
    assert(_G.a=="roberto" and _G.roberto=="a")
    _G.a = nil; _G.roberto = nil
  end

  local function f(a,b) return string.gsub(a,'.',b) end
  assert(string.gsub("trocar tudo em |teste|b| \195\169 |beleza|al|", "|([^|]*)|([^|]*)|", f) ==
              "trocar tudo em bbbbb \195\169 alalalalalal")

  local function dostring (s) return load(s, "")() or "" end
  assert(string.gsub("alo $a='x'$ novamente $return a$",
                     "$([^$]*)%$",
                     dostring) == "alo  novamente x")

  local x = string.gsub("$x=string.gsub('alo', '.', string.upper)$ assim vai para $return x$",
           "$([^$]*)%$", dostring)
  assert(x == ' assim vai para ALO')
  _G.a, _G.x = nil

  local t = {}
  local s = 'a alo jose  joao'
  local r = string.gsub(s, '()(%w+)()', function (a,w,b)
               assert(string.len(w) == b-a);
               t[a] = b-a;
             end)
  assert(s == r and t[1] == 1 and t[3] == 3 and t[7] == 4 and t[13] == 4)
end
