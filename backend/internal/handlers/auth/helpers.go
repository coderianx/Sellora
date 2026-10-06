package auth

import (
	"crypto/rand"
	"errors"
	"math/big"
	"net/mail"
)

func isValidEmail(s string) bool {
	_, err := mail.ParseAddress(s)
	return err == nil
}

func generateRefreshToken(length int) (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	if length <= 0 {
		return "", errors.New("token length must be positive")
	}

	token := make([]byte, length)

	for i := range token {
		n, err := rand.Int(
			rand.Reader,
			big.NewInt(int64(len(chars))),
		)

		if err != nil {
			return "", err
		}

		token[i] = chars[n.Int64()]
	}

	return string(token), nil
}
