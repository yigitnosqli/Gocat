-- Port Scanner Example
-- Scans a range of ports on a target host

local target_host = "localhost"
local start_port = 1
local end_port = 1000

log("info", "Starting port scan on " .. target_host)
log("info", "Scanning ports " .. start_port .. " to " .. end_port)

local open_ports = {}

for port = start_port, end_port do
    -- Try to connect to the port
    local conn, err = connect(target_host, port, "tcp")
    
    if conn then
        table.insert(open_ports, port)
        log("info", "Port " .. port .. " is OPEN")
        close(conn)
    end
    
    -- Small delay to avoid overwhelming the target
    if port % 100 == 0 then
        log("info", "Scanned " .. port .. " ports...")
        sleep(0.1)
    end
end

log("info", "Scan complete!")
log("info", "Found " .. #open_ports .. " open ports")

for _, port in ipairs(open_ports) do
    log("info", "  - Port " .. port)
end
