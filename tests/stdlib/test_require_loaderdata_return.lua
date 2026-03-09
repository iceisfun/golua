-- require should return loader data on first successful load.

do
  package.loaded.__req_loaderdata = nil
  package.searchers = {
    function(name)
      return function(modname, loaderdata)
        return "mod:" .. modname
      end, "LD"
    end,
  }

  local m1, ld1 = require("__req_loaderdata")
  assert(m1 == "mod:__req_loaderdata")
  assert(ld1 == "LD")

  local n1 = select("#", require("__req_loaderdata"))
  assert(n1 == 1, "cached call should have one result")

  package.loaded.__req_loaderdata2 = nil
  package.searchers = {
    function(name)
      return function(modname, loaderdata)
        return "mod:" .. modname
      end
    end,
  }

  local m2, ld2 = require("__req_loaderdata2")
  assert(m2 == "mod:__req_loaderdata2")
  assert(ld2 == nil)

  local n2 = select("#", require("__req_loaderdata2"))
  assert(n2 == 1, "cached call should have one result")
end
