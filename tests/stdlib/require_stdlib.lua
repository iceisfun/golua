-- Test: attrib.lua - Basic require of standard libraries
-- From: attrib.lua
-- What: Verifies that require returns the correct module tables for standard libraries (string, math, table, io, os, coroutine)

do
  assert(require"string" == string)
  assert(require"math" == math)
  assert(require"table" == table)
  assert(require"io" == io)
  assert(require"os" == os)
  assert(require"coroutine" == coroutine)
end
