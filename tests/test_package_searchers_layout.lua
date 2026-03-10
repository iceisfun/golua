do
  assert(#package.searchers == 4, "package.searchers should expose four default searchers")
  for i = 1, 4 do
    assert(type(package.searchers[i]) == "function", "searcher " .. i .. " should be a function")
  end
end

do
  local ok, err = pcall(require, "a.b.c")
  assert(not ok)
  assert(type(err) == "string")
  assert(string.find(err, "no file 'a/b/c.so'", 1, true), err)
  assert(string.find(err, "no file 'a.so'", 1, true), err)
end
