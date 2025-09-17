-- Banner Grabber Script for GoCat
-- Connects to services and grabs their banners

function grab_banner(host, port, timeout)
    timeout = timeout or 5
    
    log("info", "Grabbing banner from " .. host .. ":" .. port)
    
    local conn, err = connect(host, port, "tcp")
    if not conn then
        log("error", "Failed to connect to " .. host .. ":" .. port .. " - " .. (err or "unknown error"))
        return nil
    end
    
    -- Wait a bit for the service to send banner
    sleep(1)
    
    -- Try to receive banner
    local banner, recv_err = receive(conn, 1024)
    if banner and #banner > 0 then
        log("info", "Banner received from " .. host .. ":" .. port)
        log("info", "Banner: " .. banner)
    else
        -- Some services need a request first
        log("debug", "No initial banner, trying HTTP request...")
        send(conn, "GET / HTTP/1.0\r\n\r\n")
        sleep(1)
        banner, recv_err = receive(conn, 1024)
        
        if banner and #banner > 0 then
            log("info", "HTTP response received from " .. host .. ":" .. port)
            log("info", "Response: " .. string.sub(banner, 1, 200) .. "...")
        else
            log("warn", "No banner received from " .. host .. ":" .. port)
        end
    end
    
    close(conn)
    return banner
end

function grab_common_services(host)
    local common_ports = {
        {21, "FTP"},
        {22, "SSH"},
        {23, "Telnet"},
        {25, "SMTP"},
        {53, "DNS"},
        {80, "HTTP"},
        {110, "POP3"},
        {143, "IMAP"},
        {443, "HTTPS"},
        {993, "IMAPS"},
        {995, "POP3S"}
    }
    
    log("info", "Starting banner grabbing for common services on " .. host)
    
    for _, service in ipairs(common_ports) do
        local port = service[1]
        local name = service[2]
        
        log("info", "Checking " .. name .. " service on port " .. port)
        local banner = grab_banner(host, port, 3)
        
        if banner then
            log("info", name .. " (" .. port .. ") banner: " .. string.sub(banner, 1, 100))
        end
        
        sleep(0.5) -- Be nice to the target
    end
    
    log("info", "Banner grabbing completed for " .. host)
end

-- Main execution
local target_host = "127.0.0.1"

log("info", "GoCat Banner Grabber starting...")
grab_common_services(target_host)
log("info", "Banner grabbing finished.")