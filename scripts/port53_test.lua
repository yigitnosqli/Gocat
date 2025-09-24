-- Port 53 Specific Test Script for False Positive/Negative Check
-- Testing accuracy against nmap results

-- Configuration - Target port 53 specifically
local CONFIG = {
    host = "127.0.0.1",
    start_port = 50,    -- Include port 53
    end_port = 60,      -- Small range around 53
    delay = 0.01,       -- Fast scanning
    progress_interval = 1
}

-- Enhanced port scanner with detailed logging
function scan_port_detailed(host, port)
    log("info", "🔍 Testing " .. host .. ":" .. port .. " (nmap shows 53 should be OPEN)")
    
    local conn, err = connect(host, port, "tcp")
    if conn then
        log("info", "✅ SUCCESS: Port " .. port .. " is OPEN - Connection established")
        close(conn)
        return true
    else
        log("debug", "❌ FAILED: Port " .. port .. " connection failed - " .. (err or "refused"))
        return false
    end
end

-- Test the specific range
function test_accuracy()
    log("info", "🎯 GoCat vs nmap Accuracy Test")
    log("info", "📊 nmap result: Port 53/tcp OPEN (domain service)")
    log("info", "🔬 Testing GoCat accuracy on range " .. CONFIG.start_port .. "-" .. CONFIG.end_port)
    
    local found_ports = {}
    local total_tests = CONFIG.end_port - CONFIG.start_port + 1
    
    for port = CONFIG.start_port, CONFIG.end_port do
        if scan_port_detailed(CONFIG.host, port) then
            table.insert(found_ports, port)
            log("info", "🎉 MATCH FOUND: Port " .. port .. " confirmed OPEN")
        end
        sleep(CONFIG.delay)
    end
    
    log("info", "")
    log("info", "🔍 ACCURACY TEST RESULTS:")
    log("info", "📈 Scanned range: " .. CONFIG.start_port .. "-" .. CONFIG.end_port .. " (" .. total_tests .. " ports)")
    log("info", "🎯 Expected: Port 53 OPEN (according to nmap)")
    log("info", "📊 Found: " .. #found_ports .. " open ports")
    
    if #found_ports > 0 then
        log("info", "✅ Open ports detected: " .. table.concat(found_ports, ", "))
        
        -- Check if we found port 53
        local found_53 = false
        for _, port in ipairs(found_ports) do
            if port == 53 then
                found_53 = true
                break
            end
        end
        
        if found_53 then
            log("info", "🎉 ACCURACY: TRUE POSITIVE - Port 53 correctly detected as OPEN")
        else
            log("warn", "⚠️  POTENTIAL ISSUE: Found other ports but not port 53")
        end
    else
        log("warn", "❌ ACCURACY ISSUE: No open ports found, but nmap shows port 53 as OPEN")
        log("warn", "🔬 This could indicate a FALSE NEGATIVE")
    end
    
    return found_ports
end

-- Run the accuracy test
log("info", "🚀 Starting GoCat Port Scanner Accuracy Validation...")
local results = test_accuracy()
log("info", "🏁 Accuracy test completed. Results: " .. #results .. " open ports")