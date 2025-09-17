-- HTTP Client Script for GoCat
-- Simple HTTP client implementation

function http_request(host, port, method, path, headers, body)
    method = method or "GET"
    path = path or "/"
    headers = headers or {}
    body = body or ""
    
    log("info", "Making " .. method .. " request to " .. host .. ":" .. port .. path)
    
    local conn, err = connect(host, port, "tcp")
    if not conn then
        log("error", "Failed to connect: " .. (err or "unknown error"))
        return nil
    end
    
    -- Build HTTP request
    local request = method .. " " .. path .. " HTTP/1.1\r\n"
    request = request .. "Host: " .. host .. "\r\n"
    request = request .. "User-Agent: GoCat-Lua/1.0\r\n"
    request = request .. "Connection: close\r\n"
    
    -- Add custom headers
    for key, value in pairs(headers) do
        request = request .. key .. ": " .. value .. "\r\n"
    end
    
    -- Add content length if body exists
    if #body > 0 then
        request = request .. "Content-Length: " .. #body .. "\r\n"
    end
    
    request = request .. "\r\n" .. body
    
    log("debug", "Sending HTTP request...")
    local sent, send_err = send(conn, request)
    if not sent or sent == 0 then
        log("error", "Failed to send request: " .. (send_err or "unknown error"))
        close(conn)
        return nil
    end
    
    -- Receive response
    log("debug", "Receiving HTTP response...")
    local response = ""
    local chunk_size = 1024
    
    while true do
        local chunk, recv_err = receive(conn, chunk_size)
        if not chunk or #chunk == 0 then
            break
        end
        response = response .. chunk
        
        -- Prevent infinite loops
        if #response > 100000 then
            log("warn", "Response too large, truncating...")
            break
        end
    end
    
    close(conn)
    
    if #response > 0 then
        log("info", "Received " .. #response .. " bytes")
        return response
    else
        log("warn", "No response received")
        return nil
    end
end

function parse_http_response(response)
    if not response then
        return nil
    end
    
    local lines = {}
    for line in response:gmatch("[^\r\n]+") do
        table.insert(lines, line)
    end
    
    if #lines == 0 then
        return nil
    end
    
    -- Parse status line
    local status_line = lines[1]
    local version, status_code, reason = status_line:match("(HTTP/[%d%.]+)%s+(%d+)%s*(.*)")
    
    log("info", "HTTP Response: " .. status_code .. " " .. (reason or ""))
    
    -- Find headers and body separator
    local header_end = 1
    for i = 2, #lines do
        if lines[i] == "" then
            header_end = i
            break
        end
        log("debug", "Header: " .. lines[i])
    end
    
    return {
        version = version,
        status_code = tonumber(status_code),
        reason = reason,
        headers = table.concat(lines, "\n", 2, header_end - 1),
        body = table.concat(lines, "\n", header_end + 1)
    }
end

function simple_get(host, port, path)
    local response = http_request(host, port, "GET", path)
    return parse_http_response(response)
end

function simple_post(host, port, path, data, content_type)
    content_type = content_type or "application/x-www-form-urlencoded"
    local headers = {["Content-Type"] = content_type}
    local response = http_request(host, port, "POST", path, headers, data)
    return parse_http_response(response)
end

-- Main execution examples
log("info", "GoCat HTTP Client starting...")

-- Example 1: Simple GET request
log("info", "Example 1: GET request to localhost")
local get_response = simple_get("127.0.0.1", 80, "/")
if get_response then
    log("info", "GET Response Status: " .. get_response.status_code)
    if get_response.body and #get_response.body > 0 then
        log("info", "Body preview: " .. string.sub(get_response.body, 1, 200))
    end
end

-- Example 2: POST request
log("info", "Example 2: POST request")
local post_data = "name=test&value=123"
local post_response = simple_post("127.0.0.1", 80, "/api/test", post_data)
if post_response then
    log("info", "POST Response Status: " .. post_response.status_code)
end

log("info", "HTTP Client examples completed.")