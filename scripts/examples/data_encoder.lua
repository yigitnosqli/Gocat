-- Data Encoder Example
-- Demonstrates encoding and decoding functions

local original_text = "Hello, GoCat!"

log("info", "Original text: " .. original_text)

-- Hex encoding
local hex_encoded = hex_encode(original_text)
log("info", "Hex encoded: " .. hex_encoded)

local hex_decoded = hex_decode(hex_encoded)
log("info", "Hex decoded: " .. hex_decoded)

-- Base64 encoding
local b64_encoded = base64_encode(original_text)
log("info", "Base64 encoded: " .. b64_encoded)

local b64_decoded = base64_decode(b64_encoded)
log("info", "Base64 decoded: " .. b64_decoded)

-- Verify
if original_text == hex_decoded and original_text == b64_decoded then
    log("info", "All encoding/decoding tests passed!")
else
    log("error", "Encoding/decoding test failed!")
end
