-- Test: constructs.lua - Semicolons
-- From: constructs.lua
-- What: Tests that semicolons are valid statement separators

do
  ;;print "testing syntax";;
  local a
  do ;;; end
  ; do ; a = 3; assert(a == 3) end;
  ;
end
