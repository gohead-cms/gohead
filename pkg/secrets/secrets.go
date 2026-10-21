// Package secrets encrypts the credentials GoHead keeps for other services
// (the tokens of connected apps), with a key derived from the instance's
// jwt_secret.
//
// Without a key (jwt_secret not set, so each process would pick a different
// one), values are stored as they are: encrypting them with a key no other
// process knows, and that the next start forgets, would lose them.
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
)

// prefix marks an encrypted value.
const prefix = "enc:v1:"

// ErrUndecryptable is returned for a value encrypted with another key: the
// jwt_secret changed since it was stored.
var ErrUndecryptable = errors.New("cannot decrypt a stored secret: jwt_secret changed since it was saved; enter it again")

var (
	mu  sync.RWMutex
	gcm cipher.AEAD
)

// SetKey derives the encryption key from material (the jwt_secret). An empty
// material removes the key.
func SetKey(material string) error {
	mu.Lock()
	defer mu.Unlock()
	if material == "" {
		gcm = nil
		return nil
	}
	key, err := hkdf.Key(sha256.New, []byte(material), nil, "gohead connection secrets v1", 32)
	if err != nil {
		return err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	g, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	gcm = g
	return nil
}

// Enabled reports whether values are encrypted.
func Enabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return gcm != nil
}

// Seal returns the value to store for plain: encrypted when a key is set.
func Seal(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	mu.RLock()
	g := gcm
	mu.RUnlock()
	if g == nil {
		return plain, nil
	}
	nonce := make([]byte, g.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	out := g.Seal(nonce, nonce, []byte(plain), nil)
	return prefix + base64.RawStdEncoding.EncodeToString(out), nil
}

// Open returns the plain value of a stored one.
func Open(stored string) (string, error) {
	if !strings.HasPrefix(stored, prefix) {
		return stored, nil // stored without a key
	}
	mu.RLock()
	g := gcm
	mu.RUnlock()
	if g == nil {
		return "", ErrUndecryptable
	}
	raw, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(stored, prefix))
	if err != nil || len(raw) < g.NonceSize() {
		return "", ErrUndecryptable
	}
	plain, err := g.Open(nil, raw[:g.NonceSize()], raw[g.NonceSize():], nil)
	if err != nil {
		return "", ErrUndecryptable
	}
	return string(plain), nil
}
