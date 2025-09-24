-- Banner Grabber Script for GoCat
-- Purpose: Service fingerprinting through banner collection
-- Feature: Multi-protocol banner grabbing (FTP, SSH, HTTP, SMTP, etc.)
-- Feature: Smart protocol detection and appropriate requests
-- Feature: SSL/TLS support for secure services
-- Feature: Customizable timeout and retry mechanisms
-- Usage: Configure target host and run to identify services

-- Configuration
local CONFIG = {
    host = "127.0.0.1",        -- Target host to scan
    timeout = 5,               -- Connection timeout in seconds
    read_timeout = 2,          -- Banner read timeout
    delay = 0.5,               -- Delay between service checks
    max_banner_size = 2048,    -- Maximum banner size to read
    retry_count = 2            -- Number of retries for failed connections
}

-- Service definitions with protocols
local SERVICES = {
    {21,   "FTP",     "tcp", "220", nil},
    {22,   "SSH",     "tcp", "SSH", nil},
    {23,   "Telnet",  "tcp", nil,   nil},
    {25,   "SMTP",    "tcp", "220", nil},
    {53,   "DNS",     "tcp", nil,   nil},
    {80,   "HTTP",    "tcp", "HTTP", "GET / HTTP/1.1\r\nHost: %s\r\nUser-Agent: GoCat-BannerGrabber/2.0\r\nConnection: close\r\n\r\n"},
    {110,  "POP3",    "tcp", "+OK", nil},
    {143,  "IMAP",    "tcp", "OK",  nil},
    {443,  "HTTPS",   "ssl", "HTTP", "GET / HTTP/1.1\r\nHost: %s\r\nUser-Agent: GoCat-BannerGrabber/2.0\r\nConnection: close\r\n\r\n"},
    {993,  "IMAPS",   "ssl", "OK",  nil},
    {995,  "POP3S",   "ssl", "+OK", nil},
    {3389, "RDP",     "tcp", nil,   nil},
    {5432, "PostgreSQL", "tcp", nil, nil},
    {3306, "MySQL",   "tcp", nil,   nil}
}

-- Clean and format banner text
function clean_banner(banner)
    if not banner then return "" end
    
    -- Remove control characters and trim
    banner = string.gsub(banner, "[%c]", " ")
    banner = string.gsub(banner, "%s+", " ")
    banner = string.match(banner, "^%s*(.-)%s*$") or banner
    
    return banner
end

-- Grab banner from a specific service
function grab_banner(host, port, protocol, expected_banner, request_template, timeout)
    timeout = timeout or CONFIG.timeout
    
    if not host or not port or port <= 0 or port > 65535 then
        log("error", "Invalid host or port parameters")
        return nil
    end
    
    log("debug", "🔍 Attempting to grab banner from " .. host .. ":" .. port .. " (" .. protocol .. ")")
    
    local conn, err
    local retry_count = 0
    
    -- Retry connection if it fails
    while retry_count < CONFIG.retry_count do
        conn, err = connect(host, port, protocol)
        if conn then
            break
        end
        retry_count = retry_count + 1
        if retry_count < CONFIG.retry_count then
            log("debug", "Connection failed, retrying (" .. retry_count .. "/" .. CONFIG.retry_count .. ")")
            sleep(1)
        end
    end
    
    if not conn then
        log("debug", "❌ Failed to connect to " .. host .. ":" .. port .. " - " .. (err or "connection failed"))
        return nil
    end
    
    local banner = ""
    
    -- Wait for initial banner (many services send greeting)
    sleep(CONFIG.read_timeout)
    local initial_banner, recv_err = receive(conn, CONFIG.max_banner_size)
    
    if initial_banner and #initial_banner > 0 then
        banner = initial_banner
        log("debug", "📨 Received initial banner from " .. host .. ":" .. port)
    else
        -- If no initial banner, try sending a request
        if request_template then
            local request = string.format(request_template, host)
            log("debug", "📤 Sending request to " .. host .. ":" .. port)
            local bytes_sent, send_err = send(conn, request)
            
            if bytes_sent and bytes_sent > 0 then
                sleep(CONFIG.read_timeout)
                banner, recv_err = receive(conn, CONFIG.max_banner_size)
            else
                log("debug", "Failed to send request: " .. (send_err or "send failed"))
            end
        end
    end
    
    close(conn)
    
    if banner and #banner > 0 then
        banner = clean_banner(banner)
        log("info", "✅ Banner grabbed from " .. host .. ":" .. port)
        return banner
    else
        log("debug", "⚠️  No banner received from " .. host .. ":" .. port)
        return nil
    end
end

-- Check if a port is open before banner grabbing
function is_port_open(host, port)
    local conn, err = connect(host, port, "tcp")
    if conn then
        close(conn)
        return true
    end
    return false
end

-- Grab banners from common services
function grab_common_services(host)
    if not host or host == "" then
        log("error", "Invalid host specified")
        return {}
    end
    
    log("info", "🚀 Starting banner grabbing for common services on " .. host)
    log("info", "📋 Scanning " .. #SERVICES .. " common services")
    
    local results = {}
    local found_services = 0
    
    for i, service in ipairs(SERVICES) do
        local port = service[1]
        local name = service[2]
        local protocol = service[3] or "tcp"
        local expected = service[4]
        local request = service[5]
        
        log("info", "[" .. i .. "/" .. #SERVICES .. "] 🔍 Checking " .. name .. " on port " .. port)
        
        -- First check if port is open
        if is_port_open(host, port) then
            log("debug", "Port " .. port .. " is open, attempting banner grab")
            
            local banner = grab_banner(host, port, protocol, expected, request, CONFIG.timeout)
            
            if banner and #banner > 0 then
                found_services = found_services + 1
                local result = {
                    port = port,
                    service = name,
                    protocol = protocol,
                    banner = banner
                }
                table.insert(results, result)
                
                -- Truncate banner for display
                local display_banner = banner
                if #display_banner > 100 then
                    display_banner = string.sub(display_banner, 1, 97) .. "..."
                end
                
                log("info", "🎯 " .. name .. " (" .. port .. "/" .. protocol .. "): " .. display_banner)
            else
                log("debug", "No banner received from " .. name .. " service")
            end
        else
            log("debug", "Port " .. port .. " appears closed")
        end
        
        -- Rate limiting
        if CONFIG.delay > 0 and i < #SERVICES then
            sleep(CONFIG.delay)
        end
    end
    
    log("info", "✅ Banner grabbing completed for " .. host)
    log("info", "📊 Found " .. found_services .. " services with banners out of " .. #SERVICES .. " checked")
    
    return results
end

-- Generate summary report
function generate_report(host, results)
    if not results or #results == 0 then
        log("info", "📝 No services found to report")
        return
    end
    
    log("info", "📝 === BANNER GRABBING REPORT ===")
    log("info", "🎯 Target: " .. host)
    log("info", "📅 Scan Date: " .. os.date("%Y-%m-%d %H:%M:%S"))
    log("info", "📊 Services Found: " .. #results)
    log("info", "")
    
    for i, result in ipairs(results) do
        log("info", "[" .. i .. "] " .. result.service .. " (" .. result.port .. "/" .. result.protocol .. ")")
        log("info", "    Banner: " .. result.banner)
        log("info", "")
    end
    
    log("info", "📝 === END OF REPORT ===")
end

-- Main execution
if CONFIG.host and CONFIG.host ~= "" then
    log("info", "🚀 GoCat Banner Grabber v2.0 starting...")
    log("info", "🎯 Target: " .. CONFIG.host)
    log("info", "⏱️  Timeout: " .. CONFIG.timeout .. "s")
    
    local results = grab_common_services(CONFIG.host)
    generate_report(CONFIG.host, results)
    
    log("info", "🏁 Banner grabbing finished.")
else
    log("error", "Invalid configuration. Please set a valid host in CONFIG section.")
end
