package machine

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type GeneratedSecrets struct {
	Password string // Generic password for any machine
}

func GeneratePasswordHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

func GenerateRandomPassword() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random password: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func GenerateSHA512Crypt(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	saltString := base64.StdEncoding.EncodeToString(salt)[:16]

	hash := sha512.Sum512([]byte(password + saltString))
	hashString := base64.StdEncoding.EncodeToString(hash[:])

	return fmt.Sprintf("$6$%s$%s", saltString, hashString), nil
}
