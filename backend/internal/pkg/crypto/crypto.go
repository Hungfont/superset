package crypto

import (
	"encoding/base64"
	"errors"
	"io"
	"net/url"
	"strings"

	"superset/auth-service/internal/domain/db"

	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
)

const (
	KeySize = 32
)

func ParseEncryptionKey(encryptionKey string) ([]byte, error) {
	trimmed := strings.TrimSpace(encryptionKey)
	if len(trimmed) != KeySize {
		return nil, errors.New("encryption key must be 32 characters")
	}

	if len(trimmed) == KeySize {
		return []byte(trimmed), nil
	}

	return nil, errors.New("invalid encryption key")
}

func Encrypt(plainText string, encryptionKey []byte) (string, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	cipherText := gcm.Seal(nil, nonce, []byte(plainText), nil)
	combined := append(nonce, cipherText...)
	return base64.StdEncoding.EncodeToString(combined), nil
}

func Decrypt(encryptedText string, encryptionKey []byte) (string, error) {
	combined, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(combined) < nonceSize {
		return "", errors.New("invalid ciphertext")
	}

	nonce := combined[:nonceSize]
	cipherText := combined[nonceSize:]
	plain, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", err
	}

	return string(plain), nil
}

func DecryptSQLAlchemyURIPassword(sqlalchemyURI string, encryptionKey []byte) (string, error) {
	parsedURI, err := url.Parse(sqlalchemyURI)
	if err != nil {
		return "", err
	}

	if parsedURI.User == nil {
		return sqlalchemyURI, nil
	}

	username := parsedURI.User.Username()
	encryptedPassword, hasPassword := parsedURI.User.Password()
	if !hasPassword || encryptedPassword == "" {
		return sqlalchemyURI, nil
	}

	plainPassword, err := Decrypt(encryptedPassword, encryptionKey)
	if err != nil {
		return sqlalchemyURI, nil
	}

	parsedURI.User = url.UserPassword(username, plainPassword)
	return parsedURI.String(), nil
}

func EncryptSQLAlchemyURIPassword(sqlalchemyURI string, encryptionKey []byte) (string, error) {
	parsedURI, err := url.Parse(sqlalchemyURI)
	if err != nil {
		return "", err
	}

	if parsedURI.User == nil {
		return sqlalchemyURI, nil
	}

	username := parsedURI.User.Username()
	password, hasPassword := parsedURI.User.Password()
	if !hasPassword || password == "" {
		return sqlalchemyURI, nil
	}

	encryptedPassword, err := Encrypt(password, encryptionKey)
	if err != nil {
		return "", err
	}

	parsedURI.User = url.UserPassword(username, encryptedPassword)
	return parsedURI.String(), nil
}

func ParseSQLAlchemyURI(sqlalchemyURI string) (*url.URL, error) {
	parsedURI, err := url.Parse(sqlalchemyURI)
	if err != nil {
		return nil, db.ErrInvalidDatabaseURI
	}
	if parsedURI.Scheme == "" || parsedURI.Host == "" {
		return nil, db.ErrInvalidDatabaseURI
	}
	return parsedURI, nil
}

func MaskSQLAlchemyURI(sqlalchemyURI string) (string, error) {
	parsedURI, err := url.Parse(sqlalchemyURI)
	if err != nil {
		return "", err
	}

	if parsedURI.User == nil {
		return sqlalchemyURI, nil
	}

	username := parsedURI.User.Username()
	_, hasPassword := parsedURI.User.Password()
	if !hasPassword {
		return sqlalchemyURI, nil
	}

	parsedURI.User = url.UserPassword(username, "***")
	maskedURI := parsedURI.String()
	return strings.Replace(maskedURI, "%2A%2A%2A", "***", 1), nil
}