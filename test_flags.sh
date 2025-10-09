#!/bin/bash
# GoCat Flag Test Script

echo "==================================="
echo "🧪 GoCat Flag Tests"
echo "==================================="
echo ""

GOCAT="./gocat"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

test_count=0
pass_count=0
fail_count=0

# Test function
test_flag() {
    local test_name="$1"
    local command="$2"
    local expected_exit="$3"
    
    test_count=$((test_count + 1))
    echo -n "Test $test_count: $test_name... "
    
    if eval "$command" > /dev/null 2>&1; then
        actual_exit=0
    else
        actual_exit=$?
    fi
    
    if [ "$actual_exit" -eq "$expected_exit" ]; then
        echo -e "${GREEN}✓ PASS${NC}"
        pass_count=$((pass_count + 1))
    else
        echo -e "${RED}✗ FAIL${NC} (expected exit: $expected_exit, got: $actual_exit)"
        fail_count=$((fail_count + 1))
    fi
}

echo "📦 Basic Commands"
echo "-----------------------------------"
test_flag "Version command" "$GOCAT version" 0
test_flag "Help command" "$GOCAT --help" 0
test_flag "Connect help" "$GOCAT connect --help" 0
test_flag "Listen help" "$GOCAT listen --help" 0
test_flag "Scan help" "$GOCAT scan --help" 0

echo ""
echo "🔌 Protocol Flags"
echo "-----------------------------------"
test_flag "UDP flag (--udp)" "$GOCAT --udp --help" 0
test_flag "SCTP flag (--sctp)" "$GOCAT --sctp --help" 0
test_flag "IPv4 flag (-4)" "$GOCAT -4 --help" 0
test_flag "IPv6 flag (-6)" "$GOCAT -6 --help" 0
test_flag "SSL flag (--ssl)" "$GOCAT --ssl --help" 0

echo ""
echo "🛡️ Security Flags"
echo "-----------------------------------"
test_flag "SSL verify flag" "$GOCAT --ssl-verify --help" 0
test_flag "SSL cert flag" "$GOCAT --ssl-cert test.pem --help" 0
test_flag "SSL key flag" "$GOCAT --ssl-key test.key --help" 0

echo ""
echo "📊 Scan Command Flags"
echo "-----------------------------------"
test_flag "Scan timeout" "$GOCAT scan --scan-timeout 5s --help" 0
test_flag "Scan concurrency" "$GOCAT scan --concurrency 50 --help" 0
test_flag "Scan ports flag" "$GOCAT scan --ports 1-100 --help" 0

echo ""
echo "🚇 Tunnel Command Flags"
echo "-----------------------------------"
test_flag "SSH tunnel flag" "$GOCAT tunnel --help" 0
test_flag "Tunnel local flag" "$GOCAT tunnel --local 8080 --help" 0
test_flag "Tunnel remote flag" "$GOCAT tunnel --remote 9000 --help" 0
test_flag "Dynamic SOCKS" "$GOCAT tunnel --dynamic --help" 0

echo ""
echo "🔄 Convert Command Flags"
echo "-----------------------------------"
test_flag "Convert from flag" "$GOCAT convert --from tcp:8080 --help" 0
test_flag "Convert to flag" "$GOCAT convert --to udp:9000 --help" 0
test_flag "Convert buffer flag" "$GOCAT convert --buffer 4096 --help" 0

echo ""
echo "📁 Transfer Command Flags"
echo "-----------------------------------"
test_flag "Transfer send mode" "$GOCAT transfer send --help" 0
test_flag "Transfer receive mode" "$GOCAT transfer receive --help" 0
test_flag "Transfer progress flag" "$GOCAT transfer --progress --help" 0
test_flag "Transfer checksum flag" "$GOCAT transfer --checksum --help" 0

echo ""
echo "🌐 DNS Tunnel Flags"
echo "-----------------------------------"
test_flag "DNS tunnel server" "$GOCAT dns-tunnel --server --help" 0
test_flag "DNS tunnel client" "$GOCAT dns-tunnel --client --help" 0
test_flag "DNS tunnel domain" "$GOCAT dns-tunnel --domain test.com --help" 0

echo ""
echo "🔄 Proxy Command Flags"
echo "-----------------------------------"
test_flag "Proxy listen flag" "$GOCAT proxy --listen :8080 --help" 0
test_flag "Proxy target flag" "$GOCAT proxy --target http://backend --help" 0
test_flag "Proxy backends flag" "$GOCAT proxy --backends http://b1,http://b2 --help" 0

echo ""
echo "💬 Chat/Broker Flags"
echo "-----------------------------------"
test_flag "Chat max connections" "$GOCAT chat --max-conns 50 --help" 0
test_flag "Chat room name" "$GOCAT chat --room TestRoom --help" 0
test_flag "Broker max connections" "$GOCAT broker --max-conns 20 --help" 0

echo ""
echo "📜 Script Command Flags"
echo "-----------------------------------"
test_flag "Script run command" "$GOCAT script run --help" 0
test_flag "Script list command" "$GOCAT script list --help" 0
test_flag "Script validate command" "$GOCAT script validate --help" 0

echo ""
echo "🖥️ TUI Command"
echo "-----------------------------------"
test_flag "TUI command" "$GOCAT tui --help" 0

echo ""
echo "==================================="
echo "📊 Test Summary"
echo "==================================="
echo "Total tests: $test_count"
echo -e "${GREEN}Passed: $pass_count${NC}"
if [ $fail_count -gt 0 ]; then
    echo -e "${RED}Failed: $fail_count${NC}"
else
    echo -e "${GREEN}Failed: $fail_count${NC}"
fi
echo ""

if [ $fail_count -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed!${NC}"
    exit 1
fi
