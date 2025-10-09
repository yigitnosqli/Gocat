-- Advanced Chat Bot Example
-- Demonstrates a more sophisticated chat bot with multiple features

local config = {
    port = 8888,
    max_connections = 10,
    bot_name = "GoCat-Bot",
    welcome_message = "Welcome to GoCat Chat! Type 'help' for commands.",
    commands = {
        help = "Show available commands",
        time = "Show current time",
        echo = "Echo your message",
        joke = "Tell a random joke",
        quit = "Disconnect from chat",
        users = "Show connected users count"
    }
}

-- Simple jokes database
local jokes = {
    "Why do programmers prefer dark mode? Because light attracts bugs!",
    "How many programmers does it take to change a light bulb? None, that's a hardware problem!",
    "Why do Java developers wear glasses? Because they can't C#!",
    "What's a programmer's favorite hangout place? Foo Bar!",
    "Why did the programmer quit his job? He didn't get arrays!"
}

-- Connection counter
local connection_count = 0

-- Bot response functions
local function get_time_response()
    -- Since we don't have access to real time in Lua, return a placeholder
    return "Current server time: " .. os.date("%Y-%m-%d %H:%M:%S")
end

local function get_help_response()
    local help_text = "Available commands:\n"
    for cmd, desc in pairs(config.commands) do
        help_text = help_text .. "  /" .. cmd .. " - " .. desc .. "\n"
    end
    return help_text
end

local function get_joke_response()
    local joke_index = math.random(1, #jokes)
    return jokes[joke_index]
end

local function process_command(message)
    -- Remove leading slash and convert to lowercase
    local command = message:sub(2):lower()
    local parts = {}
    for part in command:gmatch("%S+") do
        table.insert(parts, part)
    end
    
    local cmd = parts[1]
    local args = {}
    for i = 2, #parts do
        table.insert(args, parts[i])
    end
    
    if cmd == "help" then
        return get_help_response()
    elseif cmd == "time" then
        return get_time_response()
    elseif cmd == "echo" then
        if #args > 0 then
            return "Echo: " .. table.concat(args, " ")
        else
            return "Echo: (nothing to echo)"
        end
    elseif cmd == "joke" then
        return get_joke_response()
    elseif cmd == "users" then
        return "Connected users: " .. connection_count
    elseif cmd == "quit" then
        return "Goodbye! Thanks for chatting with " .. config.bot_name .. "!"
    else
        return "Unknown command: /" .. cmd .. ". Type '/help' for available commands."
    end
end

local function generate_bot_response(user_message)
    -- Check if it's a command
    if user_message:sub(1, 1) == "/" then
        return process_command(user_message)
    end
    
    -- Simple keyword-based responses
    local message_lower = user_message:lower()
    
    if message_lower:find("hello") or message_lower:find("hi") then
        return "Hello there! How can I help you today?"
    elseif message_lower:find("how are you") then
        return "I'm doing great! Thanks for asking. How are you?"
    elseif message_lower:find("weather") then
        return "I don't have access to weather data, but I hope it's nice where you are!"
    elseif message_lower:find("name") then
        return "I'm " .. config.bot_name .. ", your friendly chat companion!"
    elseif message_lower:find("thank") then
        return "You're welcome! Happy to help!"
    elseif message_lower:find("bye") or message_lower:find("goodbye") then
        return "Goodbye! It was nice chatting with you!"
    else
        -- Default responses for unrecognized input
        local default_responses = {
            "That's interesting! Tell me more.",
            "I see. What else would you like to talk about?",
            "Hmm, that's something to think about!",
            "Thanks for sharing that with me.",
            "I'm still learning, but I find that fascinating!"
        }
        local response_index = math.random(1, #default_responses)
        return default_responses[response_index]
    end
end

local function handle_client_connection(conn)
    connection_count = connection_count + 1
    local client_id = "Client-" .. connection_count
    
    log("info", "New client connected: " .. client_id)
    
    -- Send welcome message
    local welcome = config.bot_name .. ": " .. config.welcome_message .. "\n"
    send(conn, welcome)
    
    -- Chat loop
    while true do
        -- Receive message from client
        local message, err = receive(conn, 1024)
        if err or not message or message == "" then
            log("info", "Client disconnected: " .. client_id)
            break
        end
        
        -- Clean up the message
        message = message:gsub("[\r\n]", "")
        
        if message ~= "" then
            log("info", client_id .. " says: " .. message)
            
            -- Check for quit command
            if message:lower() == "/quit" then
                local goodbye = config.bot_name .. ": " .. process_command("/quit") .. "\n"
                send(conn, goodbye)
                break
            end
            
            -- Generate and send bot response
            local response = generate_bot_response(message)
            local bot_message = config.bot_name .. ": " .. response .. "\n"
            send(conn, bot_message)
            
            -- Add a small delay to make conversation feel more natural
            sleep(0.5)
        end
    end
    
    connection_count = connection_count - 1
    close(conn)
end

-- Main chat bot function
local function start_chat_bot()
    log("info", "Starting " .. config.bot_name .. " on port " .. config.port)
    
    -- Initialize random seed
    math.randomseed(os.time())
    
    -- Start listening for connections
    local listener, err = listen(config.port, "tcp")
    if err then
        log("error", "Failed to start chat bot: " .. err)
        return
    end
    
    log("info", config.bot_name .. " is ready!")
    log("info", "Connect with: gocat connect localhost " .. config.port)
    log("info", "Maximum connections: " .. config.max_connections)
    
    -- In a real implementation, we would accept multiple connections
    -- For this example, we'll simulate handling one connection at a time
    log("info", "Waiting for connections...")
    
    -- This is a simplified example - in a real implementation with proper
    -- accept() function, we would handle multiple concurrent connections
    log("info", "Chat bot started successfully!")
    log("info", "Note: This is a demonstration of the chat bot logic.")
    log("info", "In a full implementation, this would accept and handle multiple connections.")
    
    return listener
end

-- Start the chat bot
local listener = start_chat_bot()

if listener then
    log("info", "Chat bot is running. Press Ctrl+C to stop.")
else
    log("error", "Failed to start chat bot")
end