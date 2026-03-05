-- Custom headers with http.fetch
local res = http.fetch({
    url = "https://httpbin.org/headers",
    headers = {
        ["User-Agent"] = "GoLua",
        ["Accept"] = "application/json"
    }
})
print("Status:", res.status)
local data = res.json()
print("User-Agent echoed:", data.headers["User-Agent"])
