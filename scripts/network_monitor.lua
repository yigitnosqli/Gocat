-- Network Monitor Script for GoCat
-- Monitors network connectivity and logs status

local monitor_config = {
    targets = {
        {host = "8.8.8.8", port = 53, name = "Google DNS", protocol = "tcp"},
        {host = "1.1.1.1", port = 53, name = "Cloudflare DNS", protocol = "tcp"},
        {host = "127.0.0.1", port = 22, name = "Local SSH", protocol = "tcp"},
        {host = "127.0.0.1", port = 80, name = "Local HTTP", protocol = "tcp"}
    },
    check_interval = 30, -- seconds
    timeout = 5,         -- seconds
    max_failures = 3     -- consecutive failures before alert
}

local target_status = {}

function check_connectivity(target)
    log("debug", "Checking " .. target.name .. " (" .. target.host .. ":" .. target.port .. ")")
    
    local conn, err = connect(target.host, target.port, target.protocol)
    if conn then
        close(conn)
        return true, "Connected successfully"
    else
        return false, err or "Connection failed"
    end
end

function update_target_status(target, is_up, message)
    local key = target.host .. ":" .. target.port
    
    if not target_status[key] then
        target_status[key] = {
            name = target.name,
            consecutive_failures = 0,
            last_status = nil,
            last_check = 0,
            total_checks = 0,
            total_failures = 0
        }
    end
    
    local status = target_status[key]
    status.total_checks = status.total_checks + 1
    status.last_check = os.time()
    
    if is_up then
        if status.consecutive_failures > 0 then
            log("info", "✅ " .. target.name .. " is back online!")
        end
        status.consecutive_failures = 0
        status.last_status = "UP"
    else
        status.consecutive_failures = status.consecutive_failures + 1
        status.total_failures = status.total_failures + 1
        status.last_status = "DOWN"
        
        if status.consecutive_failures == 1 then
            log("warn", "⚠️  " .. target.name .. " is down: " .. message)
        elseif status.consecutive_failures >= monitor_config.max_failures then
            log("error", "🚨 " .. target.name .. " has been down for " .. status.consecutive_failures .. " consecutive checks!")
        end
    end
end

function monitor_targets()
    log("info", "Starting network monitoring...")
    log("info", "Monitoring " .. #monitor_config.targets .. " targets every " .. monitor_config.check_interval .. " seconds")
    
    local cycle = 0
    
    while true do
        cycle = cycle + 1
        log("info", "--- Monitoring Cycle " .. cycle .. " ---")
        
        for _, target in ipairs(monitor_config.targets) do
            local is_up, message = check_connectivity(target)
            update_target_status(target, is_up, message)
            
            if is_up then
                log("debug", "✅ " .. target.name .. " - OK")
            else
                log("warn", "❌ " .. target.name .. " - " .. message)
            end
            
            -- Small delay between checks
            sleep(1)
        end
        
        -- Show summary every 10 cycles
        if cycle % 10 == 0 then
            show_monitoring_summary()
        end
        
        log("debug", "Waiting " .. monitor_config.check_interval .. " seconds until next check...")
        sleep(monitor_config.check_interval)
    end
end

function show_monitoring_summary()
    log("info", "=== Network Monitoring Summary ===")
    
    for key, status in pairs(target_status) do
        local uptime_percent = 0
        if status.total_checks > 0 then
            uptime_percent = ((status.total_checks - status.total_failures) / status.total_checks) * 100
        end
        
        local status_icon = status.last_status == "UP" and "✅" or "❌"
        
        log("info", string.format("%s %s - Status: %s, Uptime: %.1f%%, Failures: %d/%d",
            status_icon,
            status.name,
            status.last_status or "UNKNOWN",
            uptime_percent,
            status.total_failures,
            status.total_checks
        ))
    end
    
    log("info", "================================")
end

function ping_test(host, count)
    count = count or 4
    log("info", "Ping test to " .. host .. " (" .. count .. " attempts)")
    
    local successful = 0
    local total_time = 0
    
    for i = 1, count do
        local start_time = os.clock()
        local conn, err = connect(host, 80, "tcp") -- Use port 80 as a connectivity test
        local end_time = os.clock()
        
        if conn then
            close(conn)
            local response_time = (end_time - start_time) * 1000 -- Convert to milliseconds
            log("info", "Ping " .. i .. ": " .. host .. " - " .. string.format("%.2f ms", response_time))
            successful = successful + 1
            total_time = total_time + response_time
        else
            log("warn", "Ping " .. i .. ": " .. host .. " - Failed (" .. (err or "timeout") .. ")")
        end
        
        if i < count then
            sleep(1) -- Wait 1 second between pings
        end
    end
    
    log("info", "Ping statistics for " .. host .. ":")
    log("info", "  Packets: Sent = " .. count .. ", Received = " .. successful .. ", Lost = " .. (count - successful))
    
    if successful > 0 then
        local avg_time = total_time / successful
        log("info", "  Average response time: " .. string.format("%.2f ms", avg_time))
    end
    
    local loss_percent = ((count - successful) / count) * 100
    log("info", "  Packet loss: " .. string.format("%.1f%%", loss_percent))
end

-- Main execution
log("info", "GoCat Network Monitor loaded!")

-- Example usage:
log("info", "Running ping test example...")
ping_test("8.8.8.8", 3)

log("info", "To start continuous monitoring, uncomment the next line:")
log("info", "-- monitor_targets()")

-- Uncomment to start monitoring:
-- monitor_targets()