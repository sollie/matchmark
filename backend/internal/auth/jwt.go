package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	accessTokenTTL  = time.Hour
	refreshTokenTTL = 7 * 24 * time.Hour
	rawTokenBytes   = 32
	TokenIssuer     = "matchmark"
	TokenAudience   = "matchmark-api"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

type Claims struct {
	UserID   int64  `json:"uid"`
	Username string `json:"sub"`
	Role     string `json:"role"`
	AuthVer  int64  `json:"av"`
	jwt.RegisteredClaims
}

func SignAccessToken(secret string, userID int64, username, role string, authVersion int64) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		AuthVer:  authVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    TokenIssuer,
			Audience:  jwt.ClaimStrings{TokenAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

func VerifyAccessToken(secret, tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	},
		jwt.WithIssuer(TokenIssuer),
		jwt.WithAudience(TokenAudience),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func GenerateRefreshToken() (raw string, hashed string, expiresAt time.Time, err error) {
	b := make([]byte, rawTokenBytes)
	if _, err = rand.Read(b); err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate refresh token: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	hashed = HashRefreshToken(raw)
	expiresAt = time.Now().Add(refreshTokenTTL)
	return raw, hashed, expiresAt, nil
}

func HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
