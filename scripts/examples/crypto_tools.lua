-- Cryptographic Tools Script for GoCat
-- Various encryption, hashing, and encoding utilities

-- Generate secure password
function generate_password(length, options)
    length = length or 16
    options = options or {
        uppercase = true,
        lowercase = true,
        numbers = true,
        symbols = true
    }
    
    local chars = ""
    if options.lowercase then chars = chars .. "abcdefghijklmnopqrstuvwxyz" end
    if options.uppercase then chars = chars .. "ABCDEFGHIJKLMNOPQRSTUVWXYZ" end
    if options.numbers then chars = chars .. "0123456789" end
    if options.symbols then chars = chars .. "!@#$%^&*()_+-=[]{}|;:,.<>?" end
    
    if chars == "" then
        ui.error("No character set selected!")
        return nil
    end
    
    -- Generate random key and use it to select characters
    local key = crypto.generate_key(length)
    local password = ""
    
    for i = 1, length do
        local idx = (string.byte(key, i) % string.len(chars)) + 1
        password = password .. string.sub(chars, idx, idx)
    end
    
    return password
end

-- Hash file with multiple algorithms
function hash_file(filepath)
    ui.info(string.format("Hashing file: %s", filepath))
    
    local content = file.read(filepath)
    if not content then
        ui.error("Failed to read file")
        return nil
    end
    
    local hashes = {
        md5 = crypto.md5(content),
        sha1 = crypto.sha1(content),
        sha256 = crypto.sha256(content)
    }
    
    ui.cyan("\n╔══════════════════════════════════════════╗")
    ui.cyan("║              File Hashes                 ║")
    ui.cyan("╚══════════════════════════════════════════╝")
    
    print(string.format("\nFile: %s", filepath))
    print(string.format("Size: %d bytes", string.len(content)))
    print("\nHashes:")
    print(string.format("  MD5:    %s", hashes.md5))
    print(string.format("  SHA1:   %s", hashes.sha1))
    print(string.format("  SHA256: %s", hashes.sha256))
    
    return hashes
end

-- Encode/Decode utilities
function encode_decode_tool()
    ui.cyan("\n╔══════════════════════════════════════════╗")
    ui.cyan("║         Encode/Decode Utility            ║")
    ui.cyan("╚══════════════════════════════════════════╝")
    
    local test_string = "Hello, GoCat! 🚀"
    
    print(string.format("\nOriginal: %s", test_string))
    print(string.format("Length: %d bytes", string.len(test_string)))
    
    -- Base64
    local b64_encoded = crypto.base64_encode(test_string)
    local b64_decoded = crypto.base64_decode(b64_encoded)
    print(string.format("\nBase64 Encoded: %s", b64_encoded))
    print(string.format("Base64 Decoded: %s", b64_decoded))
    
    -- Hex
    local hex_encoded = crypto.hex_encode(test_string)
    local hex_decoded = crypto.hex_decode(hex_encoded)
    print(string.format("\nHex Encoded: %s", hex_encoded))
    print(string.format("Hex Decoded: %s", hex_decoded))
end

-- Password strength checker
function check_password_strength(password)
    local score = 0
    local feedback = {}
    
    -- Length check
    local length = string.len(password)
    if length >= 8 then score = score + 1 end
    if length >= 12 then score = score + 1 end
    if length >= 16 then score = score + 1 end
    
    -- Character variety
    if string.match(password, "[a-z]") then 
        score = score + 1 
    else
        table.insert(feedback, "Add lowercase letters")
    end
    
    if string.match(password, "[A-Z]") then 
        score = score + 1 
    else
        table.insert(feedback, "Add uppercase letters")
    end
    
    if string.match(password, "[0-9]") then 
        score = score + 1 
    else
        table.insert(feedback, "Add numbers")
    end
    
    if string.match(password, "[^a-zA-Z0-9]") then 
        score = score + 1 
    else
        table.insert(feedback, "Add special characters")
    end
    
    -- Common patterns check
    if string.match(password, "123") or string.match(password, "abc") then
        score = score - 1
        table.insert(feedback, "Avoid sequential characters")
    end
    
    if string.match(password, "password") or string.match(password, "admin") then
        score = score - 2
        table.insert(feedback, "Avoid common words")
    end
    
    -- Determine strength
    local strength = "Very Weak"
    local color_func = ui.red
    
    if score >= 7 then
        strength = "Very Strong"
        color_func = ui.green
    elseif score >= 5 then
        strength = "Strong"
        color_func = ui.green
    elseif score >= 3 then
        strength = "Moderate"
        color_func = ui.yellow
    elseif score >= 1 then
        strength = "Weak"
        color_func = ui.red
    end
    
    -- Display results
    print(string.format("\nPassword: %s", string.rep("*", string.len(password))))
    print(string.format("Length: %d characters", length))
    color_func(string.format("Strength: %s (Score: %d/10)", strength, score))
    
    if #feedback > 0 then
        ui.yellow("\nSuggestions:")
        for _, suggestion in ipairs(feedback) do
            print(string.format("  • %s", suggestion))
        end
    end
    
    return score, strength
end

-- Generate cryptographic keys
function generate_keys()
    ui.cyan("\n╔══════════════════════════════════════════╗")
    ui.cyan("║         Cryptographic Key Generator      ║")
    ui.cyan("╚══════════════════════════════════════════╝")
    
    -- Generate various key sizes
    local keys = {
        {size = 16, name = "AES-128"},
        {size = 24, name = "AES-192"},
        {size = 32, name = "AES-256"},
        {size = 64, name = "SHA-512 HMAC"}
    }
    
    print("\nGenerated Keys:")
    for _, key_info in ipairs(keys) do
        local key = crypto.generate_key(key_info.size)
        print(string.format("\n%s (%d bytes):", key_info.name, key_info.size))
        print(string.format("  Hex: %s", key))
        print(string.format("  Base64: %s", crypto.base64_encode(crypto.hex_decode(key))))
    end
    
    -- Save keys to file
    local filename = string.format("keys_%s.txt", os.date("%Y%m%d_%H%M%S"))
    local content = "Generated Cryptographic Keys\n"
    content = content .. string.format("Date: %s\n\n", os.date())
    
    for _, key_info in ipairs(keys) do
        local key = crypto.generate_key(key_info.size)
        content = content .. string.format("%s: %s\n", key_info.name, key)
    end
    
    file.write(filename, content)
    ui.success(string.format("\nKeys saved to %s", filename))
end

-- Main menu
function main()
    ui.cyan("╔══════════════════════════════════════════╗")
    ui.cyan("║       GoCat Crypto Tools v1.0           ║")
    ui.cyan("╚══════════════════════════════════════════╝")
    
    print("\n1. Generate secure password")
    print("2. Hash file")
    print("3. Encode/Decode demo")
    print("4. Check password strength")
    print("5. Generate cryptographic keys")
    print("6. Run all demos")
    
    -- For demo, run all
    ui.info("\nRunning all crypto demonstrations...")
    
    -- Generate password
    ui.yellow("\n=== Password Generator ===")
    local password = generate_password(16)
    ui.success(string.format("Generated password: %s", password))
    check_password_strength(password)
    
    -- Encode/Decode
    ui.yellow("\n=== Encode/Decode ===")
    encode_decode_tool()
    
    -- Generate keys
    ui.yellow("\n=== Key Generation ===")
    generate_keys()
    
    ui.success("\nAll demonstrations completed!")
end

-- Run if executed directly
if not ... then
    main()
end
