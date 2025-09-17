# GoCat Lua Scripts

This directory contains Lua scripts designed for GoCat. These scripts utilize GoCat's Lua engine feature to automate various network tasks.

## Available Scripts

### 1. `port_scanner.lua`
**Purpose:** Performs port scanning on target hosts.

**Features:**
- Scans specified port ranges
- Detects and logs open ports
- Shows progress indicators
- Rate limiting to avoid overwhelming targets

**Usage:**
```lua
-- Edit parameters within the script:
local host = "127.0.0.1"
local start_port = 20
local end_port = 100
```

### 2. `banner_grabber.lua`
**Amaç:** Servislerin banner bilgilerini toplar.

**Özellikler:**
- Yaygın servislerin bannerlarını alır (FTP, SSH, HTTP, vb.)
- HTTP request gönderebilir
- Service fingerprinting için kullanılabilir

**Kullanım:**
```lua
-- Hedef host'u değiştirin:
local target_host = "127.0.0.1"
```

### 3. `http_client.lua`
**Amaç:** Basit HTTP client implementasyonu.

**Özellikler:**
- GET ve POST requestleri
- Custom header desteği
- Response parsing
- HTTP/1.1 uyumlu

**Kullanım:**
```lua
local response = simple_get("example.com", 80, "/")
local post_response = simple_post("api.example.com", 80, "/data", "key=value")
```

### 4. `chat_bot.lua`
**Amaç:** Basit chat bot servisi.

**Özellikler:**
- Komut tabanlı yanıtlar
- Echo komutu
- Genişletilebilir response sistemi
- Chat session yönetimi

**Kullanım:**
```lua
-- Test için:
test_chat_responses()

-- Server başlatmak için:
start_chat_server(8888)
```

### 5. `network_monitor.lua`
**Amaç:** Network bağlantılarını izler.

**Özellikler:**
- Çoklu hedef izleme
- Uptime hesaplama
- Failure alerting
- Ping test fonksiyonu

**Kullanım:**
```lua
-- Ping test:
ping_test("8.8.8.8", 4)

-- Sürekli monitoring:
monitor_targets()
```

### 6. `data_encoder.lua`
**Amaç:** Veri encoding/decoding utilities.

**Özellikler:**
- Hex, Base64, URL encoding
- HTML entity encoding
- Caesar cipher (ROT13)
- Morse code encoding
- Binary encoding

**Kullanım:**
```lua
local encoded = hex_encode("Hello World")
local decoded = hex_decode(encoded)
local rot13 = caesar_cipher("Hello", 13)
```

## GoCat Lua API

Bu scriptler aşağıdaki GoCat Lua API fonksiyonlarını kullanır:

### Network Fonksiyonları
- `connect(host, port, protocol)` - Bağlantı kur
- `listen(port, protocol)` - Dinleme başlat
- `send(conn, data)` - Veri gönder
- `receive(conn, size)` - Veri al
- `close(conn)` - Bağlantıyı kapat

### Utility Fonksiyonları
- `log(level, message)` - Log mesajı
- `sleep(seconds)` - Bekle
- `hex_encode(data)` - Hex encoding
- `hex_decode(hex)` - Hex decoding
- `base64_encode(data)` - Base64 encoding
- `base64_decode(b64)` - Base64 decoding

### Environment Bilgisi
- `gocat.version` - GoCat versiyonu
- `gocat.platform` - Platform bilgisi

## Script Çalıştırma

GoCat'te Lua scriptlerini çalıştırmak için:

1. **TUI üzerinden:**
   - Script menüsüne gidin
   - İstediğiniz scripti seçin
   - Çalıştırın

2. **Komut satırından:**
   ```bash
   gocat script run port_scanner.lua
   ```

3. **Programatik olarak:**
   ```go
   engine := scripting.NewLuaEngine(nil)
   engine.LoadScript("scripts/port_scanner.lua")
   engine.ExecuteScript("port_scanner")
   ```

## Script Geliştirme

Yeni script geliştirirken:

1. **Error handling** kullanın
2. **Logging** ekleyin
3. **Rate limiting** uygulayın
4. **Configurable parameters** kullanın
5. **Documentation** ekleyin

## Güvenlik

- Scriptler sandbox modunda çalışır
- File system erişimi kısıtlıdır
- Network erişimi kontrollüdür
- Dangerous fonksiyonlar devre dışıdır

## Örnekler

### Basit Port Tarama
```lua
local open_ports = scan_range("192.168.1.1", 80, 443)
for _, port in ipairs(open_ports) do
    log("info", "Open port: " .. port)
end
```

### HTTP Health Check
```lua
local response = simple_get("api.example.com", 80, "/health")
if response and response.status_code == 200 then
    log("info", "Service is healthy")
else
    log("error", "Service is down")
end
```

### Data Encoding
```lua
local secret = "my secret data"
local encoded = base64_encode(secret)
local decoded = base64_decode(encoded)
log("info", "Original: " .. secret)
log("info", "Encoded: " .. encoded)
log("info", "Decoded: " .. decoded)
```

Bu scriptler GoCat'in güçlü Lua scripting özelliklerini gösterir ve network görevlerinizi otomatikleştirmenize yardımcı olur.