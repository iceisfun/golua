-- Simple POST request with automatic JSON encoding
local res = http.post("https://httpbin.org/post", {
    name = "alice",
    id = 42
})
print("Status:", res.status)
print("Body:", res.body)
