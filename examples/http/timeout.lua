-- Timeout override example
local ok, err = pcall(http.get, "https://httpbin.org/delay/10", { timeout = 5 })
if not ok then
    print("Request timed out as expected:", err)
else
    print("Request completed")
end
