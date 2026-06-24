-- A syntax error whose "near" token is a fully-scanned short string literal
-- shows the DECODED contents (Lua's txtToken renders the lexer buffer, which
-- holds decoded escapes plus delimiters), not the raw source escapes.
-- Previously golua printed the verbatim source slice ('"\65\66\67"').

print(pcall(load, 'local x "\\65\\66\\67"'))
--> ~near '"ABC"'$

print(pcall(load, 'local x "hello\\tworld"'))
--> ~near '"hello\tworld"'$

-- Long strings ([[...]] / [==[...]==]) have no escapes; the buffer holds them
-- verbatim, so they keep their raw form.
print(pcall(load, 'local x [[a\\tb]]'))
--> ~near '\[\[a\\tb\]\]'$
