-- Simple GET request example
local res = http.get("https://example.com")
print("Status:", res.status)
print("Body length:", #res.body)
