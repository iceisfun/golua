-- Test: tostring float coercion
-- From: strings.lua
-- What: Tests the "standard" float-to-string coercion where 0.0 formats as "0.0" (with trailing .0), and the concatenation behavior of integers vs floats.

do
if tostring(0.0) == "0.0" then   -- "standard" coercion float->string
  assert('' .. 12 == '12' and 12.0 .. '' == '12.0')
  assert(tostring(-1203 + 0.0) == "-1203.0")
else   -- compatible coercion
  assert(tostring(0.0) == "0")
  assert('' .. 12 == '12' and 12.0 .. '' == '12')
  assert(tostring(-1203 + 0.0) == "-1203")
end
end
