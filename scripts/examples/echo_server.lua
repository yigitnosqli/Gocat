-- Echo Server Example
-- Listens on a port and echoes back received data

local port = 8888

log("info", "Starting echo server on port " .. port)

-- Start listening
local listener, err = listen(port, "tcp")
if err then
    log("error", "Failed to listen: " .. err)
    return
end

log("info", "Echo server listening on port " .. port)
log("info", "Press Ctrl+C to stop")

-- Accept connections (simplified - in real use, you'd need accept() function)
-- This is a demonstration of the API
log("info", "Server started successfully")
log("info", "Use 'gocat connect localhost " .. port .. "' to test")
