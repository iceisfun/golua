# HTTP Module

The HTTP module is an **optional, non-standard** extension that allows Lua scripts to perform HTTP requests through Go's native `net/http` package.

## Enabling

The module is not available by default. The host application must explicitly enable it:

```go
import (
    "github.com/iceisfun/golua/v2/stdlib"
    gohttp "github.com/iceisfun/golua/v2/stdlib/http"
    "github.com/iceisfun/golua/v2/vm"
)

v := vm.New()
stdlib.Open(v)
gohttp.Open(v) // registers the http global
```

## Security Considerations

- HTTP access only exists if the host calls `http.Open(v)`
- All requests respect VM context cancellation
- Default timeout of 1 minute prevents runaway requests
- No global HTTP client state is shared between VMs
- Hosts can omit this module entirely for sandboxed environments

## API Reference

### http.get(url [, options])

Performs a GET request.

```lua
local res = http.get("https://example.com")
print(res.status) -- 200
print(res.body)   -- response body as string
```

With options:

```lua
local res = http.get("https://example.com", {
    headers = { ["Authorization"] = "Bearer token" },
    timeout = 10
})
```

### http.post(url, body [, options])

Performs a POST request. If `body` is a Lua table, it is automatically encoded as JSON and the `Content-Type` header is set to `application/json`.

```lua
-- Table body (auto-JSON)
local res = http.post("https://api.example.com/users", {
    name = "alice",
    age = 30
})

-- String body
local res = http.post("https://api.example.com/data", "raw string body")
```

### http.put(url, body [, options])

Performs a PUT request. Same body handling as `http.post`.

```lua
local res = http.put("https://api.example.com/users/1", {
    name = "alice updated"
})
```

### http.patch(url, body [, options])

Performs a PATCH request. Same body handling as `http.post`.

```lua
local res = http.patch("https://api.example.com/users/1", {
    name = "new name"
})
```

### http.delete(url [, options])

Performs a DELETE request.

```lua
local res = http.delete("https://api.example.com/users/1")
```

### http.head(url [, options])

Performs a HEAD request (response body will be empty).

```lua
local res = http.head("https://example.com")
print(res.headers["Content-Type"])
```

### http.options(url [, options])

Performs an OPTIONS request.

```lua
local res = http.options("https://api.example.com")
print(res.headers["Allow"])
```

### http.fetch(options)

Full request pipeline for advanced control.

```lua
local res = http.fetch({
    method = "POST",
    url = "https://api.example.com/data",
    headers = {
        ["Authorization"] = "Bearer token",
        ["X-Custom"] = "value"
    },
    body = { id = 123 },
    timeout = 30
})
```

## Request Options

| Field     | Type           | Description                              |
|-----------|----------------|------------------------------------------|
| `url`     | string         | Request URL (required for `fetch`)       |
| `method`  | string         | HTTP method (default: `"GET"` for fetch) |
| `headers` | table          | Key-value pairs for request headers      |
| `body`    | string / table | Request body                             |
| `timeout` | number         | Timeout in seconds (default: 60)         |

## Response Object

All methods return a response table:

```lua
{
    status = 200,              -- HTTP status code (integer)
    status_text = "200 OK",    -- Full status line
    body = "...",              -- Response body as string
    headers = {                -- Response headers
        ["Content-Type"] = "application/json"
    },
    json = function() ... end  -- Convenience JSON parser
}
```

### res.json()

Parses the response body as JSON and returns a Lua table:

```lua
local res = http.get("https://api.example.com/data")
local data = res.json()
print(data.name)
print(data.items[1])
```

JSON types map to Lua types:

| JSON    | Lua     |
|---------|---------|
| object  | table   |
| array   | table   |
| string  | string  |
| number  | integer or float |
| boolean | boolean |
| null    | nil     |

## Table to JSON Coercion

When a Lua table is passed as a request body, it is automatically converted to JSON:

- Sequential integer keys starting from 1 produce a JSON array
- All other tables produce a JSON object
- Nested tables are recursively converted

```lua
-- Produces JSON array: ["a", "b", "c"]
http.post(url, { "a", "b", "c" })

-- Produces JSON object: {"name": "alice", "id": 42}
http.post(url, { name = "alice", id = 42 })
```

## Timeout Behavior

- Default timeout: **60 seconds** (1 minute)
- Override per-request via the `timeout` option (in seconds)
- Timeouts derive from the VM context via `context.WithTimeout`
- If the VM context is cancelled, pending requests are also cancelled

```lua
-- This request will timeout after 5 seconds
local ok, err = pcall(http.get, "https://slow.example.com", { timeout = 5 })
if not ok then
    print("Request failed: " .. err)
end
```

## Error Handling

HTTP errors (connection failures, timeouts, invalid URLs) raise Lua errors. Use `pcall` to handle them:

```lua
local ok, err = pcall(http.get, "https://unreachable.example.com")
if not ok then
    print("Error: " .. err)
end
```

Note: HTTP error status codes (4xx, 5xx) are **not** Lua errors. They return normally with the appropriate `status` field.
