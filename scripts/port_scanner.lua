-- Port Scanner Script for GoCat
-- Scans a range of ports on a target host

function scan_port(host, port)
    log("info", "Scanning " .. host .. ":" .. port)
    
    local conn, err = connect(host, port, "tcp")
    if conn then
        log("info", "Port " .. port .. " is OPEN")
        close(conn)
        return true
    else
        log("debug", "Port " .. port .. " is closed: " .. (err or "unknown error"))
        return false
    end
end

function scan_range(host, start_port, end_port)
    log("info", "Starting port scan on " .. host .. " from " .. start_port .. " to " .. end_port)
    
    local open_ports = {}
    local total_ports = end_port - start_port + 1
    local scanned = 0
    
    for port = start_port, end_port do
        if scan_port(host, port) then
            table.insert(open_ports, port)
        end
        
        scanned = scanned + 1
        if scanned % 10 == 0 then
            log("info", "Progress: " .. scanned .. "/" .. total_ports .. " ports scanned")
        end
        
        -- Small delay to avoid overwhelming the target
        sleep(0.1)
    end
    
    log("info", "Scan completed. Found " .. #open_ports .. " open ports")
    for _, port in ipairs(open_ports) do
        log("info", "Open port: " .. port)
    end
    
    return open_ports
end

-- Main execution
local host = "127.0.0.1"
local start_port = 20
local end_port = 100

log("info", "GoCat Port Scanner starting...")
local results = scan_range(host, start_port, end_port)
log("info", "Port scan finished. Total open ports: " .. #results)