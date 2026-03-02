-- Test: Escape sequences and long strings basic
-- From: literals.lua
-- What: Tests basic equivalence between escape sequences and long string literals, including tab and newline.

do
  assert("\n\t" == [[

	]])
  assert([[

 $debug]] == "\n $debug")
  assert([[ [ ]] ~= [[ ] ]])
end
