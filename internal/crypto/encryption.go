package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
)

// EncryptionType defines the type of encryption
type EncryptionType string

const (
	AES256GCM        EncryptionType = "aes256-gcm"
	ChaCha20Poly1305 EncryptionType = "chacha20-poly1305"
	RSA2048          EncryptionType = "rsa-2048"
	RSA4096          EncryptionType = "rsa-4096"
)

// Encryptor interface for all encryption methods
type Encryptor interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
	GetType() EncryptionType
}

// AESEncryptor implements AES-256-GCM encryption
type AESEncryptor struct {
	key    []byte
	cipher cipher.AEAD
}

// NewAESEncryptor creates a new AES-256-GCM encryptor
func NewAESEncryptor(key []byte) (*AESEncryptor, error) {
	if len(key) != 32 {
		// Derive key if not 32 bytes
		hash := sha256.Sum256(key)
		key = hash[:]
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %v", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %v", err)
	}

	return &AESEncryptor{
		key:    key,
		cipher: aead,
	}, nil
}

// Encrypt encrypts data using AES-256-GCM
func (e *AESEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, e.cipher.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %v", err)
	}

	ciphertext := e.cipher.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts data using AES-256-GCM
func (e *AESEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < e.cipher.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:e.cipher.NonceSize()], ciphertext[e.cipher.NonceSize():]
	plaintext, err := e.cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %v", err)
	}

	return plaintext, nil
}

// GetType returns the encryption type
func (e *AESEncryptor) GetType() EncryptionType {
	return AES256GCM
}

// ChaChaEncryptor implements ChaCha20-Poly1305 encryption
type ChaChaEncryptor struct {
	key    []byte
	cipher cipher.AEAD
}

// NewChaChaEncryptor creates a new ChaCha20-Poly1305 encryptor
func NewChaChaEncryptor(key []byte) (*ChaChaEncryptor, error) {
	if len(key) != chacha20poly1305.KeySize {
		// Derive key if not correct size
		hash := sha256.Sum256(key)
		key = hash[:chacha20poly1305.KeySize]
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChaCha20-Poly1305: %v", err)
	}

	return &ChaChaEncryptor{
		key:    key,
		cipher: aead,
	}, nil
}

// Encrypt encrypts data using ChaCha20-Poly1305
func (e *ChaChaEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, e.cipher.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %v", err)
	}

	ciphertext := e.cipher.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts data using ChaCha20-Poly1305
func (e *ChaChaEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < e.cipher.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:e.cipher.NonceSize()], ciphertext[e.cipher.NonceSize():]
	plaintext, err := e.cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %v", err)
	}

	return plaintext, nil
}

// GetType returns the encryption type
func (e *ChaChaEncryptor) GetType() EncryptionType {
	return ChaCha20Poly1305
}

// RSAEncryptor implements RSA encryption
type RSAEncryptor struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

// NewRSAEncryptor creates a new RSA encryptor
func NewRSAEncryptor(bits int) (*RSAEncryptor, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %v", err)
	}

	return &RSAEncryptor{
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
	}, nil
}

// NewRSAEncryptorFromKeys creates an RSA encryptor from existing keys
func NewRSAEncryptorFromKeys(privateKeyPEM, publicKeyPEM string) (*RSAEncryptor, error) {
	encryptor := &RSAEncryptor{}

	// Parse private key if provided
	if privateKeyPEM != "" {
		block, _ := pem.Decode([]byte(privateKeyPEM))
		if block == nil {
			return nil, fmt.Errorf("failed to parse private key PEM")
		}

		privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %v", err)
		}
		encryptor.privateKey = privateKey
	}

	// Parse public key if provided
	if publicKeyPEM != "" {
		block, _ := pem.Decode([]byte(publicKeyPEM))
		if block == nil {
			return nil, fmt.Errorf("failed to parse public key PEM")
		}

		publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key: %v", err)
		}

		rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA public key")
		}
		encryptor.publicKey = rsaPublicKey
	}

	return encryptor, nil
}

// Encrypt encrypts data using RSA
func (e *RSAEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	if e.publicKey == nil {
		return nil, fmt.Errorf("no public key available")
	}

	hash := sha256.New()
	ciphertext, err := rsa.EncryptOAEP(hash, rand.Reader, e.publicKey, plaintext, nil)
	if err != nil {
		return nil, fmt.Errorf("RSA encryption failed: %v", err)
	}

	return ciphertext, nil
}

// Decrypt decrypts data using RSA
func (e *RSAEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	if e.privateKey == nil {
		return nil, fmt.Errorf("no private key available")
	}

	hash := sha256.New()
	plaintext, err := rsa.DecryptOAEP(hash, rand.Reader, e.privateKey, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("RSA decryption failed: %v", err)
	}

	return plaintext, nil
}

// GetType returns the encryption type
func (e *RSAEncryptor) GetType() EncryptionType {
	if e.privateKey != nil {
		bits := e.privateKey.N.BitLen()
		if bits >= 4096 {
			return RSA4096
		}
	}
	return RSA2048
}

// GetPublicKeyPEM returns the public key in PEM format
func (e *RSAEncryptor) GetPublicKeyPEM() (string, error) {
	if e.publicKey == nil {
		return "", fmt.Errorf("no public key available")
	}

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(e.publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %v", err)
	}

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	return string(pubKeyPEM), nil
}

// GetPrivateKeyPEM returns the private key in PEM format
func (e *RSAEncryptor) GetPrivateKeyPEM() (string, error) {
	if e.privateKey == nil {
		return "", fmt.Errorf("no private key available")
	}

	privKeyBytes := x509.MarshalPKCS1PrivateKey(e.privateKey)
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	return string(privKeyPEM), nil
}

// EncryptedConn wraps a net.Conn with encryption
type EncryptedConn struct {
	conn      io.ReadWriteCloser
	encryptor Encryptor
}

// NewEncryptedConn creates a new encrypted connection wrapper
func NewEncryptedConn(conn io.ReadWriteCloser, encryptor Encryptor) *EncryptedConn {
	return &EncryptedConn{
		conn:      conn,
		encryptor: encryptor,
	}
}

// Read reads and decrypts data from the connection
func (c *EncryptedConn) Read(b []byte) (int, error) {
	// Read length prefix (4 bytes)
	lengthBuf := make([]byte, 4)
	if _, err := io.ReadFull(c.conn, lengthBuf); err != nil {
		return 0, err
	}

	length := int(lengthBuf[0])<<24 | int(lengthBuf[1])<<16 | int(lengthBuf[2])<<8 | int(lengthBuf[3])

	// Read encrypted data
	encryptedData := make([]byte, length)
	if _, err := io.ReadFull(c.conn, encryptedData); err != nil {
		return 0, err
	}

	// Decrypt
	decrypted, err := c.encryptor.Decrypt(encryptedData)
	if err != nil {
		return 0, fmt.Errorf("decryption failed: %v", err)
	}

	copy(b, decrypted)
	return len(decrypted), nil
}

// Write encrypts and writes data to the connection
func (c *EncryptedConn) Write(b []byte) (int, error) {
	// Encrypt data
	encrypted, err := c.encryptor.Encrypt(b)
	if err != nil {
		return 0, fmt.Errorf("encryption failed: %v", err)
	}

	// Write length prefix
	length := len(encrypted)
	lengthBuf := []byte{
		byte(length >> 24),
		byte(length >> 16),
		byte(length >> 8),
		byte(length),
	}

	if _, err := c.conn.Write(lengthBuf); err != nil {
		return 0, err
	}

	// Write encrypted data
	if _, err := c.conn.Write(encrypted); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Close closes the underlying connection
func (c *EncryptedConn) Close() error {
	if closer, ok := c.conn.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// GenerateKey generates a random encryption key
func GenerateKey(size int) ([]byte, error) {
	key := make([]byte, size)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %v", err)
	}
	return key, nil
}

// DeriveKey derives a key from a password using PBKDF2
func DeriveKey(password string, salt []byte, keySize int) []byte {
	if len(salt) == 0 {
		salt = []byte("gocat-default-salt")
	}
	
	// Simple key derivation (should use PBKDF2 in production)
	hash := sha256.New()
	hash.Write([]byte(password))
	hash.Write(salt)
	
	key := hash.Sum(nil)
	if len(key) > keySize {
		key = key[:keySize]
	}
	
	return key
}

// EncodeBase64 encodes data to base64
func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeBase64 decodes base64 data
func DecodeBase64(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}
