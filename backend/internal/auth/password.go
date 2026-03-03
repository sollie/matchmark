package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2Time     = 1
	argon2Memory   = 64 * 1024
	argon2Threads  = 4
	argon2KeyLen   = 32
	saltLen        = 16
	minPasswordLen = 16
)

var ErrPasswordTooShort = fmt.Errorf("password must be at least %d characters", minPasswordLen)
var ErrInvalidHash = errors.New("invalid password hash format")
var ErrPasswordMismatch = errors.New("password does not match")

func HashPassword(password string) (string, error) {
	if len(password) < minPasswordLen {
		return "", ErrPasswordTooShort
	}

	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	encoded := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2Memory,
		argon2Time,
		argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
	return encoded, nil
}

func VerifyPassword(password, encoded string) error {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return ErrInvalidHash
	}

	var m, t, p uint32
	params := parts[3]
	for _, kv := range strings.Split(params, ",") {
		pair := strings.SplitN(kv, "=", 2)
		if len(pair) != 2 {
			return ErrInvalidHash
		}
		val, err := strconv.ParseUint(pair[1], 10, 32)
		if err != nil {
			return ErrInvalidHash
		}
		switch pair[0] {
		case "m":
			m = uint32(val)
		case "t":
			t = uint32(val)
		case "p":
			p = uint32(val)
		}
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return ErrInvalidHash
	}

	storedKey, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return ErrInvalidHash
	}

	key := argon2.IDKey([]byte(password), salt, t, m, uint8(p), uint32(len(storedKey)))

	if subtle.ConstantTimeCompare(key, storedKey) != 1 {
		return ErrPasswordMismatch
	}
	return nil
}
