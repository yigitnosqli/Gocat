-- HTTP Client Example
-- Makes a simple HTTP GET request

local host = "example.com"
local port = 80
local path = "/"

log("info", "Connecting to " .. host .. ":" .. port)

-- Connect to the server
local conn, err = connect(host, port, "tcp")
if err then
    log("error", "Failed to connect: " .. err)
    return
end

log("info", "Connected successfully")

-- Send HTTP GET request
local request = "GET " .. path .. " HTTP/1.1\r\n" ..
                "Host: " .. host .. "\r\n" ..
                "User-Agent: GoCat-Lua/1.0\r\n" ..
                "Connection: close\r\n" ..
                "\r\n"

local bytes_sent, send_err = send(conn, request)
if send_err then
    log("error", "Failed to send request: " .. send_err)
    close(conn)
    return
end

log("info", "Sent " .. bytes_sent .. " bytes")

-- Receive response
log("info", "Receiving response...")
local response = ""
while true do
    local data, recv_err = receive(conn, 4096)
    if recv_err or data == "" then
        break
    end
    response = response .. data
end

log("info", "Received " .. #response .. " bytes")
log("info", "Response:\n" .. response)

-- Close connection
close(conn)
log("info", "Connection closed")
