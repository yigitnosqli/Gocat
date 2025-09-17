-- Chat Bot Script for GoCat
-- Simple chat server that responds to messages

local responses = {
    ["hello"] = "Hello! How can I help you today?",
    ["hi"] = "Hi there! Welcome to GoCat chat!",
    ["help"] = "Available commands: hello, hi, time, echo <message>, quit",
    ["time"] = "I don't have access to system time, but you're chatting with GoCat!",
    ["quit"] = "Goodbye! Thanks for chatting!",
    ["bye"] = "See you later!",
    ["gocat"] = "GoCat is an awesome network Swiss Army knife!",
    ["lua"] = "This bot is powered by Lua scripting in GoCat!",
    ["default"] = "I'm not sure how to respond to that. Type 'help' for available commands."
}

function process_message(message)
    message = string.lower(string.gsub(message, "^%s*(.-)%s*$", "%1")) -- trim whitespace
    
    -- Handle echo command
    if string.sub(message, 1, 4) == "echo" then
        local echo_text = string.sub(message, 6) -- remove "echo "
        if #echo_text > 0 then
            return "Echo: " .. echo_text
        else
            return "Echo: (empty message)"
        end
    end
    
    -- Look for exact matches first
    if responses[message] then
        return responses[message]
    end
    
    -- Look for partial matches
    for key, response in pairs(responses) do
        if string.find(message, key) then
            return response
        end
    end
    
    return responses["default"]
end

function chat_session(conn)
    log("info", "Starting chat session")
    
    -- Send welcome message
    local welcome = "Welcome to GoCat Chat Bot!\nType 'help' for commands or 'quit' to exit.\n> "
    send(conn, welcome)
    
    while true do
        -- Receive message from client
        local message, err = receive(conn, 1024)
        if not message or #message == 0 then
            if err then
                log("debug", "Client disconnected: " .. err)
            else
                log("debug", "Client disconnected")
            end
            break
        end
        
        -- Clean up the message
        message = string.gsub(message, "[\r\n]", "")
        log("info", "Received: " .. message)
        
        -- Check for quit command
        if string.lower(message) == "quit" or string.lower(message) == "exit" then
            send(conn, responses["quit"] .. "\n")
            break
        end
        
        -- Process message and send response
        local response = process_message(message)
        log("info", "Responding: " .. response)
        send(conn, response .. "\n> ")
    end
    
    log("info", "Chat session ended")
end

function start_chat_server(port)
    port = port or 8888
    
    log("info", "Starting GoCat Chat Bot server on port " .. port)
    
    local listener, err = listen(port, "tcp")
    if not listener then
        log("error", "Failed to start server: " .. (err or "unknown error"))
        return
    end
    
    log("info", "Chat bot server listening on port " .. port)
    log("info", "Connect with: telnet localhost " .. port)
    
    -- Note: This is a simplified example
    -- In a real implementation, you'd handle multiple connections
    -- For now, we'll just handle one connection at a time
    
    while true do
        log("info", "Waiting for client connection...")
        
        -- Accept connection (this would need to be implemented in the Lua API)
        -- For now, we'll simulate with a simple message
        log("info", "Chat bot ready. Use GoCat's connect feature to test.")
        sleep(10) -- Wait 10 seconds before next iteration
    end
end

-- Utility function for testing chat responses
function test_chat_responses()
    log("info", "Testing chat bot responses...")
    
    local test_messages = {
        "hello",
        "help",
        "echo test message",
        "what is gocat",
        "time please",
        "random message",
        "quit"
    }
    
    for _, msg in ipairs(test_messages) do
        local response = process_message(msg)
        log("info", "Input: '" .. msg .. "' -> Response: '" .. response .. "'")
    end
    
    log("info", "Chat bot response testing completed.")
end

-- Main execution
log("info", "GoCat Chat Bot script loaded!")

-- Run tests
test_chat_responses()

-- To start the server, uncomment the next line:
-- start_chat_server(8888)