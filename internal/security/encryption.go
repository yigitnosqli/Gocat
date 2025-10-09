package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/pbkdf2"
)

var (
	// ErrInvalidKey is returned when the encryption key is invalid
	ErrInvalidKey = errors.New("invalid encryption key")
	// ErrInvalidCiphertext is returned when decryption fails
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
	// ErrInvalidNonce is returned when the nonce is invalid
	ErrInvalidNonce = errors.New("invalid nonce")
)

// EncryptionAlgorithm represents the encryption algorithm to use
type EncryptionAlgorithm string

const (
	// AlgorithmAES256GCM uses AES-256-GCM encryption
	AlgorithmAES256GCM EncryptionAlgorithm = "aes-256-gcm"
	// AlgorithmChaCha20Poly1305 uses ChaCha20-Poly1305 encryption
	AlgorithmChaCha20Poly1305 EncryptionAlgorithm = "chacha20-poly1305"
)

// Encryptor provides encryption and decryption functionality
type Encryptor struct {
	algorithm EncryptionAlgorithm
	key       []byte
	aead      cipher.AEAD
}

// NewEncryptor creates a new encryptor with the specified algorithm and key
func NewEncryptor(algorithm EncryptionAlgorithm, key []byte) (*Encryptor, error) {
	if len(key) == 0 {
		return nil, ErrInvalidKey
	}

	e := &Encryptor{
		algorithm: algorithm,
	}

	// Derive a proper key from the provided key material
	e.key = deriveKey(key, 32) // 256 bits

	var err error
	switch algorithm {
	case AlgorithmAES256GCM:
		block, err := aes.NewCipher(e.key)
		if err != nil {
			return nil, fmt.Errorf("failed to create AES cipher: %w", err)
		}
		e.aead, err = cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("failed to create GCM: %w", err)
		}

	case AlgorithmChaCha20Poly1305:
		e.aead, err = chacha20poly1305.New(e.key)
		if err != nil {
			return nil, fmt.Errorf("failed to create ChaCha20-Poly1305 cipher: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported encryption algorithm: %s", algorithm)
	}

	return e, nil
}

// NewEncryptorFromPassword creates an encryptor using a password
func NewEncryptorFromPassword(algorithm EncryptionAlgorithm, password string, salt []byte) (*Encryptor, error) {
	if password == "" {
		return nil, errors.New("password cannot be empty")
	}

	// If no salt provided, generate one
	if len(salt) == 0 {
		salt = make([]byte, 32)
		if _, err := rand.Read(salt); err != nil {
			return nil, fmt.Errorf("failed to generate salt: %w", err)
		}
	}

	// Derive key from password using PBKDF2
	key := pbkdf2.Key([]byte(password), salt, 100000, 32, sha256.New)

	return NewEncryptor(algorithm, key)
}

// Encrypt encrypts plaintext and returns ciphertext
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	if e.aead == nil {
		return nil, errors.New("encryptor not initialized")
	}

	// Generate a random nonce
	nonce := make([]byte, e.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the plaintext
	ciphertext := e.aead.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

// Decrypt decrypts ciphertext and returns plaintext
func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	if e.aead == nil {
		return nil, errors.New("encryptor not initialized")
	}

	nonceSize := e.aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrInvalidCiphertext
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt the ciphertext
	plaintext, err := e.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCiphertext, err)
	}

	return plaintext, nil
}

// EncryptToBase64 encrypts plaintext and returns base64-encoded ciphertext
func (e *Encryptor) EncryptToBase64(plaintext []byte) (string, error) {
	ciphertext, err := e.Encrypt(plaintext)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptFromBase64 decrypts base64-encoded ciphertext
func (e *Encryptor) DecryptFromBase64(encoded string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}
	return e.Decrypt(ciphertext)
}

// GetAlgorithm returns the encryption algorithm being used
func (e *Encryptor) GetAlgorithm() EncryptionAlgorithm {
	return e.algorithm
}

// deriveKey derives a key of the specified length from input key material
func deriveKey(keyMaterial []byte, length int) []byte {
	// Use SHA-256 to derive a key of the desired length
	hash := sha256.Sum256(keyMaterial)
	if length <= 32 {
		return hash[:length]
	}

	// For longer keys, use PBKDF2
	return pbkdf2.Key(keyMaterial, hash[:], 10000, length, sha256.New)
}

// GenerateKey generates a random encryption key
func GenerateKey(length int) ([]byte, error) {
	if length <= 0 {
		return nil, errors.New("key length must be positive")
	}

	key := make([]byte, length)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	return key, nil
}

// GenerateSalt generates a random salt for key derivation
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// StreamEncryptor provides streaming encryption/decryption
type StreamEncryptor struct {
	encryptor *Encryptor
	chunkSize int
}

// NewStreamEncryptor creates a new stream encryptor
func NewStreamEncryptor(encryptor *Encryptor, chunkSize int) *StreamEncryptor {
	if chunkSize <= 0 {
		chunkSize = 64 * 1024 // 64KB default
	}

	return &StreamEncryptor{
		encryptor: encryptor,
		chunkSize: chunkSize,
	}
}

// EncryptStream encrypts data in chunks
func (se *StreamEncryptor) EncryptStream(plaintext []byte) ([]byte, error) {
	var result []byte

	for i := 0; i < len(plaintext); i += se.chunkSize {
		end := i + se.chunkSize
		if end > len(plaintext) {
			end = len(plaintext)
		}

		chunk := plaintext[i:end]
		encrypted, err := se.encryptor.Encrypt(chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt chunk: %w", err)
		}

		// Prepend chunk length (4 bytes)
		chunkLen := uint32(len(encrypted))
		result = append(result,
			byte(chunkLen>>24),
			byte(chunkLen>>16),
			byte(chunkLen>>8),
			byte(chunkLen),
		)
		result = append(result, encrypted...)
	}

	return result, nil
}

// DecryptStream decrypts chunked data
func (se *StreamEncryptor) DecryptStream(ciphertext []byte) ([]byte, error) {
	var result []byte
	offset := 0

	for offset < len(ciphertext) {
		// Read chunk length
		if offset+4 > len(ciphertext) {
			return nil, errors.New("invalid stream format: incomplete chunk length")
		}

		chunkLen := uint32(ciphertext[offset])<<24 |
			uint32(ciphertext[offset+1])<<16 |
			uint32(ciphertext[offset+2])<<8 |
			uint32(ciphertext[offset+3])
		offset += 4

		// Read chunk data
		if offset+int(chunkLen) > len(ciphertext) {
			return nil, errors.New("invalid stream format: incomplete chunk data")
		}

		chunk := ciphertext[offset : offset+int(chunkLen)]
		offset += int(chunkLen)

		decrypted, err := se.encryptor.Decrypt(chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt chunk: %w", err)
		}

		result = append(result, decrypted...)
	}

	return result, nil
}
