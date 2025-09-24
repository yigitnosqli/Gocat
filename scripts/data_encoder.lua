-- Data Encoder/Decoder Script for GoCat
-- Provides various encoding and decoding utilities

function url_encode(str)
    if not str then return "" end
    
    str = string.gsub(str, "\n", "\r\n")
    str = string.gsub(str, "([^%w %-%_%.%~])",
        function(c) return string.format("%%%02X", string.byte(c)) end)
    str = string.gsub(str, " ", "+")
    return str
end

function url_decode(str)
    if not str then return "" end
    
    str = string.gsub(str, "+", " ")
    str = string.gsub(str, "%%(%x%x)",
        function(h) return string.char(tonumber(h, 16)) end)
    str = string.gsub(str, "\r\n", "\n")
    return str
end

function html_encode(str)
    if not str then return "" end
    
    local html_entities = {
        ["&"] = "&amp;",
        ["<"] = "&lt;",
        [">"] = "&gt;",
        ['"'] = "&quot;",
        ["'"] = "&#39;"
    }
    
    for char, entity in pairs(html_entities) do
        str = string.gsub(str, char, entity)
    end
    
    return str
end

function html_decode(str)
    if not str then return "" end
    
    local html_entities = {
        ["&amp;"] = "&",
        ["&lt;"] = "<",
        ["&gt;"] = ">",
        ["&quot;"] = '"',
        ["&#39;"] = "'"
    }
    
    for entity, char in pairs(html_entities) do
        str = string.gsub(str, entity, char)
    end
    
    return str
end

function caesar_cipher(text, shift)
    if not text then return "" end
    shift = shift or 13 -- Default ROT13
    
    local result = ""
    for i = 1, #text do
        local char = string.sub(text, i, i)
        local byte = string.byte(char)
        
        if byte >= 65 and byte <= 90 then -- A-Z
            byte = ((byte - 65 + shift) % 26) + 65
        elseif byte >= 97 and byte <= 122 then -- a-z
            byte = ((byte - 97 + shift) % 26) + 97
        end
        
        result = result .. string.char(byte)
    end
    
    return result
end

function reverse_string(str)
    if not str then return "" end
    return string.reverse(str)
end

function binary_encode(str)
    if not str then return "" end
    
    local result = ""
    for i = 1, #str do
        local byte = string.byte(str, i)
        local binary = ""
        
        for j = 7, 0, -1 do
            local bit = math.floor(byte / (2^j)) % 2
            binary = binary .. bit
        end
        
        result = result .. binary .. " "
    end
    
    return string.sub(result, 1, -2) -- Remove trailing space
end

function binary_decode(str)
    if not str then return "" end
    
    local result = ""
    for binary in string.gmatch(str, "%S+") do
        if #binary == 8 then
            local byte = 0
            for i = 1, 8 do
                local bit = tonumber(string.sub(binary, i, i))
                if bit then
                    byte = byte * 2 + bit
                end
            end
            result = result .. string.char(byte)
        end
    end
    
    return result
end

function morse_encode(text)
    if not text then return "" end
    
    local morse_code = {
        A = ".-", B = "-...", C = "-.-.", D = "-..", E = ".", F = "..-.",
        G = "--.", H = "....", I = "..", J = ".---", K = "-.-", L = ".-..",
        M = "--", N = "-.", O = "---", P = ".--.", Q = "--.-", R = ".-.",
        S = "...", T = "-", U = "..-", V = "...-", W = ".--", X = "-..-",
        Y = "-.--", Z = "--..",
        ["0"] = "-----", ["1"] = ".----", ["2"] = "..---", ["3"] = "...--",
        ["4"] = "....-", ["5"] = ".....", ["6"] = "-....", ["7"] = "--...",
        ["8"] = "---..", ["9"] = "----.",
        [" "] = "/"
    }
    
    local result = ""
    for i = 1, #text do
        local char = string.upper(string.sub(text, i, i))
        if morse_code[char] then
            result = result .. morse_code[char] .. " "
        end
    end
    
    return string.sub(result, 1, -2) -- Remove trailing space
end

function test_encoders()
    log("info", "Testing GoCat Data Encoders...")
    
    local test_string = "Hello World! 123"
    log("info", "Original: " .. test_string)
    
    -- Test Hex encoding
    local hex_encoded = hex_encode(test_string)
    local hex_decoded = hex_decode(hex_encoded)
    log("info", "Hex: " .. hex_encoded .. " -> " .. hex_decoded)
    
    -- Test Base64 encoding
    local b64_encoded = base64_encode(test_string)
    local b64_decoded = base64_decode(b64_encoded)
    log("info", "Base64: " .. b64_encoded .. " -> " .. b64_decoded)
    
    -- Test URL encoding
    local url_encoded = url_encode(test_string)
    local url_decoded = url_decode(url_encoded)
    log("info", "URL: " .. url_encoded .. " -> " .. url_decoded)
    
    -- Test HTML encoding
    local html_test = "<script>alert('test');</script>"
    local html_encoded = html_encode(html_test)
    local html_decoded = html_decode(html_encoded)
    log("info", "HTML: " .. html_encoded .. " -> " .. html_decoded)
    
    -- Test Caesar cipher (ROT13)
    local caesar_encoded = caesar_cipher(test_string, 13)
    local caesar_decoded = caesar_cipher(caesar_encoded, -13)
    log("info", "ROT13: " .. caesar_encoded .. " -> " .. caesar_decoded)
    
    -- Test reverse
    local reversed = reverse_string(test_string)
    log("info", "Reverse: " .. reversed)
    
    -- Test Morse code
    local morse_encoded = morse_encode("SOS")
    log("info", "Morse 'SOS': " .. morse_encoded)
    
    log("info", "Encoder testing completed!")
end

function interactive_encoder()
    log("info", "GoCat Interactive Encoder/Decoder")
    log("info", "Available functions:")
    log("info", "  hex_encode(text), hex_decode(hex)")
    log("info", "  base64_encode(text), base64_decode(b64)")
    log("info", "  url_encode(text), url_decode(url)")
    log("info", "  html_encode(text), html_decode(html)")
    log("info", "  caesar_cipher(text, shift)")
    log("info", "  reverse_string(text)")
    log("info", "  morse_encode(text)")
    log("info", "Use these functions in your Lua scripts!")
end

-- Main execution
log("info", "GoCat Data Encoder/Decoder loaded!")

-- Run tests
test_encoders()

-- Show available functions
interactive_encoder()