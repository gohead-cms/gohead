// pkg/auth/jwt_test.go
package auth

import (
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
)

func TestJWTFunctions(t *testing.T) {
	// Initialize the JWT secret
	InitializeJWT("test-secret")

	// Define test data
	username := "testuser"
	role := "admin"

	// Test GenerateJWT
	t.Run("GenerateJWT", func(t *testing.T) {
		token, err := GenerateJWT(username, role)
		assert.NoError(t, err, "JWT generation should not return an error")
		assert.NotEmpty(t, token, "Generated JWT should not be empty")
	})

	// Test ParseJWT with a valid token
	t.Run("ParseJWT - Valid Token", func(t *testing.T) {
		token, err := GenerateJWT(username, role)
		assert.NoError(t, err)

		claims, err := ParseJWT(token)
		assert.NoError(t, err, "Parsing a valid JWT should not return an error")
		assert.Equal(t, username, claims.Username, "Username should match the claims")
		assert.Equal(t, role, claims.Role, "Role should match the claims")
		assert.WithinDuration(t, time.Now().Add(72*time.Hour), time.Unix(claims.ExpiresAt, 0), time.Minute, "Expiration time should be within the expected range")
	})

	// Test ParseJWT with an invalid token
	t.Run("ParseJWT - Invalid Token", func(t *testing.T) {
		invalidToken := "invalid-token"
		claims, err := ParseJWT(invalidToken)
		assert.Error(t, err, "Parsing an invalid JWT should return an error")
		assert.Nil(t, claims, "Claims should be nil for an invalid token")
	})

	// Test ParseJWT with an expired token
	t.Run("ParseJWT - Expired Token", func(t *testing.T) {
		// Generate a token with a past expiration time
		expiredTime := time.Now().Add(-1 * time.Hour)
		claims := &Claims{
			Username: username,
			Role:     role,
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: expiredTime.Unix(),
				IssuedAt:  time.Now().Add(-2 * time.Hour).Unix(),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, err := token.SignedString(jwtKey)
		assert.NoError(t, err)

		parsedClaims, err := ParseJWT(tokenStr)
		assert.Error(t, err, "Parsing an expired JWT should return an error")
		assert.Nil(t, parsedClaims, "Claims should be nil for an expired token")
	})
}
