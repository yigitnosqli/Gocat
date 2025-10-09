package security

import (
	"bytes"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	tests := []struct {
		name      string
		algorithm EncryptionAlgorithm
		plaintext string
	}{
		{
			name:      "AES-256-GCM short text",
			algorithm: AlgorithmAES256GCM,
			plaintext: "Hello, World!",
		},
		{
			name:      "AES-256-GCM long text",
			algorithm: AlgorithmAES256GCM,
			plaintext: "This is a longer text that should be encrypted and decrypted successfully using AES-256-GCM encryption algorithm.",
		},
		{
			name:      "ChaCha20-Poly1305 short text",
			algorithm: AlgorithmChaCha20Poly1305,
			plaintext: "Hello, World!",
		},
		{
			name:      "ChaCha20-Poly1305 long text",
			algorithm: AlgorithmChaCha20Poly1305,
			plaintext: "This is a longer text that should be encrypted and decrypted successfully using ChaCha20-Poly1305 encryption algorithm.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate a random key
			key, err := GenerateKey(32)
			if err != nil {
				t.Fatalf("Failed to generate key: %v", err)
			}

			// Create encryptor
			encryptor, err := NewEncryptor(tt.algorithm, key)
			if err != nil {
				t.Fatalf("Failed to create encryptor: %v", err)
			}

			// Encrypt
			ciphertext, err := encryptor.Encrypt([]byte(tt.plaintext))
			if err != nil {
				t.Fatalf("Failed to encrypt: %v", err)
			}

			// Decrypt
			decrypted, err := encryptor.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Failed to decrypt: %v", err)
			}

			// Verify
			if string(decrypted) != tt.plaintext {
				t.Errorf("Decrypted text doesn't match original.\nExpected: %s\nGot: %s", tt.plaintext, string(decrypted))
			}
		})
	}
}

func TestEncryptDecryptBase64(t *testing.T) {
	key, _ := GenerateKey(32)
	encryptor, _ := NewEncryptor(AlgorithmAES256GCM, key)

	plaintext := "Test message for base64 encoding"

	// Encrypt to base64
	encoded, err := encryptor.EncryptToBase64([]byte(plaintext))
	if err != nil {
		t.Fatalf("Failed to encrypt to base64: %v", err)
	}

	// Decrypt from base64
	decrypted, err := encryptor.DecryptFromBase64(encoded)
	if err != nil {
		t.Fatalf("Failed to decrypt from base64: %v", err)
	}

	if string(decrypted) != plaintext {
		t.Errorf("Decrypted text doesn't match original")
	}
}

func TestEncryptorFromPassword(t *testing.T) {
	password := "my-secret-password"
	salt, _ := GenerateSalt()

	encryptor, err := NewEncryptorFromPassword(AlgorithmAES256GCM, password, salt)
	if err != nil {
		t.Fatalf("Failed to create encryptor from password: %v", err)
	}

	plaintext := "Secret message"
	ciphertext, _ := encryptor.Encrypt([]byte(plaintext))
	decrypted, _ := encryptor.Decrypt(ciphertext)

	if string(decrypted) != plaintext {
		t.Errorf("Decrypted text doesn't match original")
	}
}

func TestStreamEncryption(t *testing.T) {
	key, _ := GenerateKey(32)
	encryptor, _ := NewEncryptor(AlgorithmAES256GCM, key)
	streamEncryptor := NewStreamEncryptor(encryptor, 1024)

	// Create a large plaintext
	plaintext := bytes.Repeat([]byte("This is a test message. "), 1000)

	// Encrypt stream
	ciphertext, err := streamEncryptor.EncryptStream(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt stream: %v", err)
	}

	// Decrypt stream
	decrypted, err := streamEncryptor.DecryptStream(ciphertext)
	if err != nil {
		t.Fatalf("Failed to decrypt stream: %v", err)
	}

	// Verify
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Decrypted stream doesn't match original")
	}
}

func TestInvalidDecryption(t *testing.T) {
	key, _ := GenerateKey(32)
	encryptor, _ := NewEncryptor(AlgorithmAES256GCM, key)

	// Try to decrypt invalid data
	_, err := encryptor.Decrypt([]byte("invalid ciphertext"))
	if err == nil {
		t.Error("Expected error when decrypting invalid data")
	}
}

func TestGenerateKey(t *testing.T) {
	lengths := []int{16, 24, 32}

	for _, length := range lengths {
		key, err := GenerateKey(length)
		if err != nil {
			t.Errorf("Failed to generate key of length %d: %v", length, err)
		}

		if len(key) != length {
			t.Errorf("Generated key has wrong length. Expected: %d, Got: %d", length, len(key))
		}
	}
}

func BenchmarkEncryptAES256GCM(b *testing.B) {
	key, _ := GenerateKey(32)
	encryptor, _ := NewEncryptor(AlgorithmAES256GCM, key)
	plaintext := []byte("Benchmark test message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encryptor.Encrypt(plaintext)
	}
}

func BenchmarkDecryptAES256GCM(b *testing.B) {
	key, _ := GenerateKey(32)
	encryptor, _ := NewEncryptor(AlgorithmAES256GCM, key)
	plaintext := []byte("Benchmark test message")
	ciphertext, _ := encryptor.Encrypt(plaintext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encryptor.Decrypt(ciphertext)
	}
}

func BenchmarkEncryptChaCha20(b *testing.B) {
	key, _ := GenerateKey(32)
	encryptor, _ := NewEncryptor(AlgorithmChaCha20Poly1305, key)
	plaintext := []byte("Benchmark test message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encryptor.Encrypt(plaintext)
	}
}
