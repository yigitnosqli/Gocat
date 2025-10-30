-- Advanced Port Scanner with Banner Grabbing
-- Uses GoCat's modular Lua API

-- Configuration
local config = {
    timeout = 3,
    threads = 10,
    verbose = true
}

-- Common ports to scan
local common_ports = {
    21,    -- FTP
    22,    -- SSH
    23,    -- Telnet
    25,    -- SMTP
    53,    -- DNS
    80,    -- HTTP
    110,   -- POP3
    143,   -- IMAP
    443,   -- HTTPS
    445,   -- SMB
    3306,  -- MySQL
    3389,  -- RDP
    5432,  -- PostgreSQL
    6379,  -- Redis
    8080,  -- HTTP Alt
    8443,  -- HTTPS Alt
    27017  -- MongoDB
}

-- Service signatures
local service_signatures = {
    ["SSH"] = "SSH-",
    ["HTTP"] = "HTTP/",
    ["FTP"] = "220",
    ["SMTP"] = "220",
    ["MySQL"] = "mysql",
    ["Redis"] = "-ERR",
    ["MongoDB"] = "MongoDB"
}

-- Scan a single port
function scan_port(host, port)
    local conn = net.connect(host, port, "tcp")
    
    if conn then
        if config.verbose then
            ui.green(string.format("[+] %s:%d - OPEN", host, port))
        end
        
        -- Try banner grabbing
        local banner = net.banner_grab(host, port)
        if banner and banner ~= "" then
            local service = identify_service(banner)
            ui.cyan(string.format("    Banner: %s", banner:sub(1, 50)))
            if service then
                ui.yellow(string.format("    Service: %s", service))
            end
        end
        
        net.close(conn)
        return true
    else
        if config.verbose then
            ui.red(string.format("[-] %s:%d - CLOSED", host, port))
        end
        return false
    end
end

-- Identify service from banner
function identify_service(banner)
    for service, signature in pairs(service_signatures) do
        if string.find(banner, signature) then
            return service
        end
    end
    return nil
end

-- Scan multiple ports
function scan_host(host, ports)
    ui.info(string.format("Scanning %s...", host))
    
    local open_ports = {}
    local start_time = time.now()
    
    for i, port in ipairs(ports) do
        ui.progress(i, #ports, string.format("Scanning port %d", port))
        
        if scan_port(host, port) then
            table.insert(open_ports, port)
        end
        
        -- Small delay to avoid overwhelming the target
        time.sleep(0.1)
    end
    
    local elapsed = time.since(start_time)
    
    -- Summary
    ui.success(string.format("\nScan completed in %.2f seconds", elapsed))
    ui.info(string.format("Found %d open ports out of %d scanned", #open_ports, #ports))
    
    if #open_ports > 0 then
        ui.green("\nOpen ports:")
        for _, port in ipairs(open_ports) do
            print(string.format("  - %d", port))
        end
    end
    
    return open_ports
end

-- Parse port range (e.g., "1-100" or "80,443,8080")
function parse_ports(port_spec)
    local ports = {}
    
    -- Check if it's a range
    local start_port, end_port = string.match(port_spec, "(%d+)-(%d+)")
    if start_port and end_port then
        for port = tonumber(start_port), tonumber(end_port) do
            table.insert(ports, port)
        end
    else
        -- Check if it's a comma-separated list
        for port in string.gmatch(port_spec, "(%d+)") do
            table.insert(ports, tonumber(port))
        end
    end
    
    -- If no ports specified, use common ports
    if #ports == 0 then
        ports = common_ports
    end
    
    return ports
end

-- Main function
function main(args)
    -- Parse arguments
    local host = args and args[1] or "127.0.0.1"
    local port_spec = args and args[2] or nil
    
    -- Header
    ui.cyan("╔══════════════════════════════════════════╗")
    ui.cyan("║        Advanced Port Scanner v1.0        ║")
    ui.cyan("╚══════════════════════════════════════════╝")
    print()
    
    -- Parse ports
    local ports = port_spec and parse_ports(port_spec) or common_ports
    
    -- Perform scan
    local results = scan_host(host, ports)
    
    -- Save results to file
    if #results > 0 then
        local filename = string.format("scan_%s_%s.txt", host, os.date("%Y%m%d_%H%M%S"))
        local content = string.format("Scan Results for %s\n", host)
        content = content .. string.format("Date: %s\n", os.date())
        content = content .. string.format("Open Ports: %s\n", table.concat(results, ", "))
        
        file.write(filename, content)
        ui.info(string.format("Results saved to %s", filename))
    end
    
    return results
end

-- Run if executed directly
if not ... then
    main()
end
