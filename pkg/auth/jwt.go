// pkg/auth/jwt.go
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenTTL is how long an issued token stays valid.
const TokenTTL = 72 * time.Hour

var jwtKey []byte

// ErrInvalidToken is returned for any token that fails parsing or validation.
var ErrInvalidToken = errors.New("invalid token")

func InitializeJWT(secret string) {
	jwtKey = []byte(secret)
}

type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`

	jwt.RegisteredClaims
}

func GenerateJWT(username, role string) (string, error) {
	now := time.Now()
	claims := &Claims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   username,
			ExpiresAt: jwt.NewNumericDate(now.Add(TokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

// ParseJWT validates signature, algorithm (HS256 only) and expiry. It never
// returns nil claims together with a nil error.
func ParseJWT(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// DerivedKey returns a key for another use of the JWT secret (signing the
// state of a sign-in with a provider): tokens signed with it are not valid
// session tokens.
func DerivedKey(purpose string) []byte {
	mac := hmac.New(sha256.New, jwtKey)
	mac.Write([]byte("gohead:" + purpose))
	return mac.Sum(nil)
}
