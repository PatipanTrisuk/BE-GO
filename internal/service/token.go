package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

func GenerateRefreshToken() (string, error) {
	randomBytes := make([]byte, 32)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(randomBytes), nil
}

func HashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))

	return hex.EncodeToString(hash[:])
}
