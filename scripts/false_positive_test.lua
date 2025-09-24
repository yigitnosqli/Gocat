-- False Positive Test Script
-- Testing ports that are definitely closed to check for false positives

-- Configuration - Test ports that should be closed
local CONFIG = {
    host = "127.0.0.1",
    test_ports = {12345, 54321, 9999, 8888, 7777, 6666}, -- Uncommon ports likely to be closed
    delay = 0.01
}

function test_false_positives()
    log("info", "🔬 FALSE POSITIVE TEST")
    log("info", "📊 Testing ports that should be CLOSED to detect false positives")
    log("info", "🎯 Expected result: ALL ports should be CLOSED")
    
    local false_positives = {}
    local total_tested = #CONFIG.test_ports
    
    for i, port in ipairs(CONFIG.test_ports) do
        log("info", "🔍 [" .. i .. "/" .. total_tested .. "] Testing " .. CONFIG.host .. ":" .. port .. " (should be CLOSED)")
        
        local conn, err = connect(CONFIG.host, port, "tcp")
        if conn then
            -- This would be a false positive!
            log("error", "🚨 FALSE POSITIVE: Port " .. port .. " reported as OPEN but should be CLOSED!")
            table.insert(false_positives, port)
            close(conn)
        else
            log("info", "✅ CORRECT: Port " .. port .. " correctly detected as CLOSED (" .. (err or "refused") .. ")")
        end
        
        sleep(CONFIG.delay)
    end
    
    log("info", "")
    log("info", "🔍 FALSE POSITIVE TEST RESULTS:")
    log("info", "📈 Total ports tested: " .. total_tested)
    log("info", "🎯 Expected closed: " .. total_tested)
    log("info", "📊 False positives found: " .. #false_positives)
    
    if #false_positives == 0 then
        log("info", "🎉 EXCELLENT: No false positives detected!")
        log("info", "✅ All tested ports correctly identified as CLOSED")
        log("info", "🔒 Scanner accuracy: 100%")
    else
        log("error", "🚨 FALSE POSITIVES DETECTED:")
        for _, port in ipairs(false_positives) do
            log("error", "  ❌ Port " .. port .. " incorrectly reported as OPEN")
        end
        local accuracy = ((total_tested - #false_positives) / total_tested) * 100
        log("warn", "📉 Scanner accuracy: " .. string.format("%.1f%%", accuracy))
    end
    
    return false_positives
end

-- Additional test with random high ports
function test_random_high_ports()
    log("info", "")
    log("info", "🎲 RANDOM HIGH PORT TEST")
    log("info", "📊 Testing random high ports (30000-65000) for false positives")
    
    local random_ports = {}
    for i = 1, 5 do
        local port = math.random(30000, 65000)
        table.insert(random_ports, port)
    end
    
    local false_positives = {}
    
    for i, port in ipairs(random_ports) do
        log("info", "🔍 Testing random port " .. port)
        
        local conn, err = connect(CONFIG.host, port, "tcp")
        if conn then
            log("warn", "⚠️  UNEXPECTED: Random port " .. port .. " is actually OPEN")
            table.insert(false_positives, port)
            close(conn)
        else
            log("info", "✅ Expected: Port " .. port .. " is closed")
        end
        
        sleep(CONFIG.delay)
    end
    
    log("info", "🎲 Random port test: " .. (#random_ports - #false_positives) .. "/" .. #random_ports .. " correctly identified as closed")
    return false_positives
end

-- Run comprehensive false positive tests
log("info", "🚀 Starting GoCat False Positive Detection Tests...")

local fp1 = test_false_positives()
local fp2 = test_random_high_ports()

local total_false_positives = #fp1 + #fp2

log("info", "")
log("info", "🏁 COMPREHENSIVE FALSE POSITIVE TEST COMPLETED")
log("info", "📊 Total false positives detected: " .. total_false_positives)

if total_false_positives == 0 then
    log("info", "🎉 PERFECT SCORE: GoCat port scanner shows NO false positives!")
    log("info", "✅ Scanner is highly accurate and reliable")
else
    log("warn", "⚠️  False positives detected: " .. total_false_positives)
    log("info", "🔧 Consider investigating connection logic or timeout settings")
end