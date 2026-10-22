package service

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory  = 19 * 1024
	argonTime    = 2
	argonThreads = 1
	argonKeyLen  = 32
	argonSaltLen = 16
)

func derive(password string, salt []byte, memory, time uint32, threads uint8) []byte {
	return argon2.IDKey([]byte(password), salt, time, memory, threads, argonKeyLen)
}

func encodeHash(salt []byte, memory, time uint32, threads uint8, key []byte) string {
	enc := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, memory, time, threads, enc.EncodeToString(salt), enc.EncodeToString(key))
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	return encodeHash(salt, argonMemory, argonTime, argonThreads, derive(password, salt, argonMemory, argonTime, argonThreads)), nil
}

type parsedHash struct {
	memory, time uint32
	threads      uint8
	salt, key    []byte
}

func parseHash(phc string) (parsedHash, bool) {
	var p parsedHash
	parts := strings.Split(phc, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return p, false
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return p, false
	}
	var threads uint32
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.time, &threads); err != nil || threads == 0 || threads > 255 {
		return p, false
	}
	p.threads = uint8(threads)
	enc := base64.RawStdEncoding
	var err error
	if p.salt, err = enc.DecodeString(parts[4]); err != nil {
		return p, false
	}
	if p.key, err = enc.DecodeString(parts[5]); err != nil || len(p.key) == 0 {
		return p, false
	}
	return p, true
}

func verifyPassword(password, phc string) (ok, needsRehash bool) {
	p, valid := parseHash(phc)
	if !valid {
		return false, false
	}
	key := argon2.IDKey([]byte(password), p.salt, p.time, p.memory, p.threads, uint32(len(p.key)))
	ok = subtle.ConstantTimeCompare(key, p.key) == 1
	needsRehash = p.memory < argonMemory || p.time < argonTime || len(p.key) < argonKeyLen
	return ok, needsRehash
}
