-- Banner Grabber Example
-- Connects to services and grabs their banners

local targets = {
    {host = "localhost", port = 22, name = "SSH"},
    {host = "localhost", port = 80, name = "HTTP"},
    {host = "localhost", port = 25, name = "SMTP"},
    {host = "localhost", port = 21, name = "FTP"},
}

log("info", "Starting banner grabbing...")

for _, target in ipairs(targets) do
    log("info", "Checking " .. target.name .. " on " .. target.host .. ":" .. target.port)
    
    local conn, err = connect(target.host, target.port, "tcp")
    if conn then
        log("info", "Connected to " .. target.name)
        
        -- Wait a bit for banner
        sleep(0.5)
        
        -- Try to receive banner
        local banner, recv_err = receive(conn, 1024)
        if banner and banner ~= "" then
            log("info", "Banner: " .. banner)
        else
            log("warn", "No banner received")
        end
        
        close(conn)
    else
        log("warn", "Could not connect to " .. target.name .. ": " .. (err or "unknown error"))
    end
    
    sleep(0.2)
end

log("info", "Banner grabbing complete")
