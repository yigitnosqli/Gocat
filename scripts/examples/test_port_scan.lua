-- Test port scanning functionality
local net = require("net")

print("Testing port scanning...")

-- Test scanning common ports
print("\n🔍 Scanning localhost common ports (80,443,8080)...")
local open_ports = net.scan("localhost", "80,443,8080")

print("Open ports found:")
for i = 1, #open_ports do
    print("  ✓ Port " .. open_ports[i] .. " is open")
end

-- Test scanning a range
print("\n🔍 Scanning localhost port range (8000-8010)...")
local range_ports = net.scan("localhost", "8000-8010")

print("Open ports in range:")
if #range_ports == 0 then
    print("  No open ports found")
else
    for i = 1, #range_ports do
        print("  ✓ Port " .. range_ports[i] .. " is open")
    end
end

print("\n✅ Port scan test completed!")
