# Directive Loader Example

Demonstrates the realistic embedder pattern for `directives`:

1. Read each `.lua` file in a directory.
2. Parse only the header (cheap — no compilation).
3. Apply embedder policy: skip `@disabled`, capture `@scope`, parse
   `@tick` as a `time.Duration`.
4. Compile and run the surviving scripts.

The `directives` package itself has no opinion about which keys mean
what. All policy lives in this example, where it belongs.

## Non-standard Lua

Header directives (`-- @key value`) are a **golua-specific extension**.
They are **not** part of the Lua language as specified by Lua 5.4 /
Lua 5.5 and are not implemented by the reference Lua interpreter.
Source files using directives remain valid, portable Lua and execute
identically on any Lua 5.4+ implementation; only the directive
*interpretation* (this example) is golua-specific.

## Run

```bash
mkdir -p /tmp/directive_loader_demo
cat > /tmp/directive_loader_demo/a.lua <<'EOF'
-- @scope worker
-- @tick 5s
print("a ran")
EOF
cat > /tmp/directive_loader_demo/b.lua <<'EOF'
-- @disabled
print("b ran")  -- never executes
EOF
cat > /tmp/directive_loader_demo/c.lua <<'EOF'
-- @scope reporter
-- @tick 1m
print("c ran")
EOF

go run ./examples/directive_loader /tmp/directive_loader_demo
```

## Output

```
loaded:   a.lua        scope=worker       tick=5s
a ran
disabled: b.lua
loaded:   c.lua        scope=reporter     tick=1m0s
c ran
```
