-- Port Scanner Script for GoCat
-- Purpose: Performs comprehensive port scanning on target hosts
-- Feature: TCP connection-based port detection
-- Feature: Configurable scan ranges and timing
-- Feature: Progress reporting and result summary
-- Feature: Rate limiting to prevent network flooding
-- Usage: Modify the configuration section below and run the script

-- Configuration
local CONFIG = {
    host = "127.0.0.1",        -- Target host to scan
    start_port = 20,           -- Starting port number
    end_port = 100,            -- Ending port number
    delay = 0.1,               -- Delay between scans (seconds)
    timeout = 3,               -- Connection timeout (seconds)
    progress_interval = 10     -- Progress report interval
}

-- Scan a single port on the target host
function scan_port(host, port)
    if not host or not port or port <= 0 or port > 65535 then
        log("error", "Invalid host or port: " .. (host or "nil") .. ":" .. (port or "nil"))
        return false
    end
    
    log("debug", "Scanning " .. host .. ":" .. port)
    
    local conn, err = connect(host, port, "tcp")
    if conn then
        log("info", "✅ Port " .. port .. " is OPEN")
        close(conn)
        return true
    else
        log("debug", "❌ Port " .. port .. " is closed: " .. (err or "connection failed"))
        return false
    end
end

-- Scan a range of ports
function scan_range(host, start_port, end_port, options)
    options = options or {}
    local delay = options.delay or CONFIG.delay
    local progress_interval = options.progress_interval or CONFIG.progress_interval
    
    -- Validation
    if not host or host == "" then
        log("error", "Host cannot be empty")
        return {}
    end
    
    if start_port <= 0 or end_port <= 0 or start_port > end_port then
        log("error", "Invalid port range: " .. start_port .. "-" .. end_port)
        return {}
    end
    
    if end_port > 65535 then
        log("warn", "End port > 65535, limiting to 65535")
        end_port = 65535
    end
    
    log("info", "🎯 Starting port scan on " .. host .. " (" .. start_port .. "-" .. end_port .. ")")
    
    local open_ports = {}
    local total_ports = end_port - start_port + 1
    local scanned = 0
    local start_time = os.time()
    
    for port = start_port, end_port do
        if scan_port(host, port) then
            table.insert(open_ports, port)
        end
        
        scanned = scanned + 1
        
        -- Progress reporting
        if scanned % progress_interval == 0 or scanned == total_ports then
            local progress_pct = math.floor((scanned / total_ports) * 100)
            log("info", "📊 Progress: " .. scanned .. "/" .. total_ports .. " (" .. progress_pct .. "%) - Found " .. #open_ports .. " open ports")
        end
        
        -- Rate limiting
        if delay > 0 and scanned < total_ports then
            sleep(delay)
        end
    end
    
    local elapsed_time = os.time() - start_time
    log("info", "✅ Scan completed in " .. elapsed_time .. " seconds")
    log("info", "📋 Found " .. #open_ports .. " open ports out of " .. total_ports .. " scanned")
    
    if #open_ports > 0 then
        log("info", "🔓 Open ports: " .. table.concat(open_ports, ", "))
    else
        log("info", "🔒 No open ports found")
    end
    
    return open_ports
end

-- Scan common ports
function scan_common_ports(host)
    local common_ports = {21, 22, 23, 25, 53, 80, 110, 143, 443, 993, 995, 3389, 5432, 3306}
    
    log("info", "🎯 Scanning common ports on " .. host)
    
    local open_ports = {}
    for _, port in ipairs(common_ports) do
        if scan_port(host, port) then
            table.insert(open_ports, port)
        end
        sleep(0.05) -- Faster for common ports
    end
    
    return open_ports
end

-- Main execution
if CONFIG.host and CONFIG.start_port and CONFIG.end_port then
    log("info", "🚀 GoCat Port Scanner v2.0 starting...")
    log("info", "🎯 Target: " .. CONFIG.host)
    log("info", "📡 Range: " .. CONFIG.start_port .. "-" .. CONFIG.end_port)
    
    local results = scan_range(CONFIG.host, CONFIG.start_port, CONFIG.end_port, CONFIG)
    
    log("info", "🏁 Port scan finished. Total open ports: " .. #results)
else
    log("error", "Invalid configuration. Please check CONFIG section.")
end
