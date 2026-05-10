package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
)

func NewToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func NewPIN() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(100000))
	if err != nil {
		return "", fmt.Errorf("generate pin: %w", err)
	}
	return fmt.Sprintf("%05d", n.Int64()), nil
}
