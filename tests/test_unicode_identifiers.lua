-- Test Unicode (UTF-8) identifiers in variables, functions, and table keys

-- Pure Unicode variable name
local 你好 = "hello"
assert(你好 == "hello", "Unicode variable name failed")

-- Mixed ASCII and Unicode variable name
local l世u界a = "lua world"
assert(l世u界a == "lua world", "Mixed ASCII/Unicode variable name failed")

-- Unicode function name with Unicode parameter name
function a测b试c001(d参e数f002)
  return d参e数f002
end
assert(a测b试c001("中国") == "中国", "Unicode function/param name failed")

-- Unicode table keys and variable name
local 测试table = {测试a = 123, 测试b = "测试b"}
assert(测试table.测试a == 123, "Unicode table dot access (number) failed")
assert(测试table.测试b == "测试b", "Unicode table dot access (string) failed")

-- Unicode keys via bracket access
assert(测试table["测试a"] == 123, "Unicode table bracket access failed")

-- Unicode in local function names
local function 计算(甲, 乙)
  return 甲 + 乙
end
assert(计算(3, 4) == 7, "Unicode local function name failed")

-- Unicode in for loop variable
local 总和 = 0
for 索引 = 1, 5 do
  总和 = 总和 + 索引
end
assert(总和 == 15, "Unicode for loop variable failed")

-- Unicode in table constructor shorthand (key = variable name)
local 名字 = "golua"
local 信息 = {名字 = 名字, 版本 = 1}
assert(信息.名字 == "golua", "Unicode table constructor key failed")
assert(信息.版本 == 1, "Unicode table constructor key (number value) failed")
