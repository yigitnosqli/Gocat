-- Test HTTP download functionality
local http = require("http")

print("Testing HTTP download...")

-- Test downloading a file
local success, result = http.download("https://httpbin.org/robots.txt", "/tmp/test_download.txt")

if success then
    print("✅ Download successful: " .. result)
else
    print("❌ Download failed: " .. result)
end
