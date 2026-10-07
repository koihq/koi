package koidomain

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 1
	argonMemory  = 32 * 1024
	argonThreads = 1
	argonKeyLen  = 32
	argonSaltLen = 16
)

// Dummy hash so Login always pays argon2, including unknown emails.
var dummyPasswordHash = mustHashPassword()

func mustHashPassword() string {
	h, err := HashPassword("not-a-real-password")
	if err != nil {
		panic(err)
	}

	return h
}

func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	return fmt.Sprintf("argon2id$v=19$m=%d$t=%d$p=%d$%s$%s",
		argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func CheckPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 7 || parts[0] != "argon2id" {
		return false
	}

	var memory, timeCost uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[2], "m=%d", &memory); err != nil {
		return false
	}

	if _, err := fmt.Sscanf(parts[3], "t=%d", &timeCost); err != nil {
		return false
	}

	if _, err := fmt.Sscanf(parts[4], "p=%d", &threads); err != nil {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	want, err := base64.RawStdEncoding.DecodeString(parts[6])
	if err != nil {
		return false
	}

	got := argon2.IDKey([]byte(password), salt, timeCost, memory, threads, uint32(len(want)))

	return subtle.ConstantTimeCompare(want, got) == 1
}
