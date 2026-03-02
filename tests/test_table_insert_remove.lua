-- Test: nextvar.lua - table.insert and table.remove
-- From: nextvar.lua
-- What: Tests table.insert and table.remove with various positions, boundary checks, and error cases for out-of-range indices.

do
  local function test (a)
    assert(not pcall(table.insert, a, 2, 20));
    table.insert(a, 10); table.insert(a, 2, 20);
    table.insert(a, 1, -1); table.insert(a, 40);
    table.insert(a, #a+1, 50)
    table.insert(a, 2, -2)
    assert(a[2] ~= undef)
    assert(a["2"] == undef)
    assert(not pcall(table.insert, a, 0, 20));
    assert(not pcall(table.insert, a, #a + 2, 20));
    assert(table.remove(a,1) == -1)
    assert(table.remove(a,1) == -2)
    assert(table.remove(a,1) == 10)
    assert(table.remove(a,1) == 20)
    assert(table.remove(a,1) == 40)
    assert(table.remove(a,1) == 50)
    assert(table.remove(a,1) == nil)
    assert(table.remove(a) == nil)
    assert(table.remove(a, #a) == nil)
  end

  a = {n=0, [-7] = "ban"}
  test(a)
  assert(a.n == 0 and a[-7] == "ban")

  a = {[-7] = "ban"};
  test(a)
  assert(a.n == nil and #a == 0 and a[-7] == "ban")

  a = {[-1] = "ban"}
  test(a)
  assert(#a == 0 and table.remove(a) == nil and a[-1] == "ban")

  a = {[0] = "ban"}
  assert(#a == 0 and table.remove(a) == "ban" and a[0] == undef)

  table.insert(a, 1, 10); table.insert(a, 1, 20); table.insert(a, 1, -1)
  assert(table.remove(a) == 10)
  assert(table.remove(a) == 20)
  assert(table.remove(a) == -1)
  assert(table.remove(a) == nil)

  a = {'c', 'd'}
  table.insert(a, 3, 'a')
  table.insert(a, 'b')
  assert(table.remove(a, 1) == 'c')
  assert(table.remove(a, 1) == 'd')
  assert(table.remove(a, 1) == 'a')
  assert(table.remove(a, 1) == 'b')
  assert(table.remove(a, 1) == nil)
  assert(#a == 0 and a.n == nil)

  a = {10,20,30,40}
  assert(table.remove(a, #a + 1) == nil)
  assert(not pcall(table.remove, a, 0))
  assert(a[#a] == 40)
  assert(table.remove(a, #a) == 40)
  assert(a[#a] == 30)
  assert(table.remove(a, 2) == 20)
  assert(a[#a] == 30 and #a == 2)
end
