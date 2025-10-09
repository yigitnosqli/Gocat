-- SSL/TLS Client Example
-- Connects to an HTTPS server

local host = "www.google.com"
local port = 443

log("info", "Connecting to " .. host .. ":" .. port .. " with SSL/TLS")

-- Connect with SSL
local conn, err = connect(host, port, "ssl")
if err then
    log("error", "Failed to connect: " .. err)
    return
end

log("info", "SSL connection established")

-- Send HTTPS request
local request = "GET / HTTP/1.1\r\n" ..
                "Host: " .. host .. "\r\n" ..
                "User-Agent: GoCat-Lua-SSL/1.0\r\n" ..
                "Connection: close\r\n" ..
                "\r\n"

send(conn, request)
log("info", "Request sent")

-- Receive response headers
local response, recv_err = receive(conn, 4096)
if response then
    -- Extract just the status line and first few headers
    local lines = {}
    for line in response:gmatch("[^\r\n]+") do
        table.insert(lines, line)
        if #lines >= 10 then break end
    end
    
    log("info", "Response headers:")
    for _, line in ipairs(lines) do
        log("info", "  " .. line)
    end
end

close(conn)
log("info", "Connection closed")
