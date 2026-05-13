# HTTP Example

Non-standard extension for HTTP requests via Go's `net/http`.

## Functions

| Function | Description |
|---|---|
| `http.get(url [, opts])` | GET request |
| `http.post(url, body [, opts])` | POST request (table body auto-encodes as JSON) |
| `http.put(url, body [, opts])` | PUT request |
| `http.patch(url, body [, opts])` | PATCH request |
| `http.delete(url [, opts])` | DELETE request |
| `http.head(url [, opts])` | HEAD request |
| `http.options(url [, opts])` | OPTIONS request |
| `http.fetch(opts)` | Full request pipeline |

## Response

All methods return a table:

```lua
{
    status = 200,
    status_text = "200 OK",
    body = "...",
    headers = { ["Content-Type"] = "application/json" },
    json = function() ... end  -- parse body as JSON
}
```

## JSON Coercion

Table bodies are automatically encoded as JSON:

```lua
-- sends {"name":"alice","id":42}
http.post(url, { name = "alice", id = 42 })

-- sends ["a","b","c"]
http.post(url, { "a", "b", "c" })
```

## Timeouts

Default timeout is 60 seconds. Override per-request:

```lua
http.get(url, { timeout = 5 })  -- 5 second timeout
```

## Security

The `http` table is **absent by default**. It only appears when the host
explicitly calls `http.Open(v)` after `stdlib.Open(v)`. This is a separate
module, not part of the standard library.

## Usage

```go
import (
    "github.com/iceisfun/golua/v1/stdlib"
    gohttp "github.com/iceisfun/golua/v1/stdlib/http"
    "github.com/iceisfun/golua/v1/vm"
)

v := vm.New()
stdlib.Open(v)
gohttp.Open(v)
```

## Running

These examples require network access to external services. They also require a
Go host that enables the optional HTTP module, so use the example runner instead
of `cmd/lua`:

```bash
go run ./examples/http ./examples/http/simple_get.lua
go run ./examples/http ./examples/http/simple_post.lua
go run ./examples/http ./examples/http/headers.lua
go run ./examples/http ./examples/http/timeout.lua
```
