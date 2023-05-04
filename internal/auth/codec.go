// Package auth contains auth settings.
package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/gofrs/uuid"

	"github.com/Yandex-Practicum/go-musthave-shortener-trainer/internal/config"
)

// EncodeUID encode uid
func EncodeUID(uid uuid.UUID) ([]byte, error) {
	c, err := aes.NewCipher(config.AuthSecret)
	if err != nil {
		return nil, fmt.Errorf("cannot create new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, fmt.Errorf("cannot create gcm from cipher: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("cannot populate nonce: %w", err)
	}

	cyphertext := gcm.Seal(nonce, nonce, uid.Bytes(), nil)
	return cyphertext, nil
}

// EncodeUIDToHex encode uid to hex
func EncodeUIDToHex(uid uuid.UUID) (string, error) {
	b, err := EncodeUID(uid)
	if err != nil {
		return "", fmt.Errorf("cannot encode uid: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// DecodeUID decode uid to hex
func DecodeUID(ciphertext []byte) (*uuid.UUID, error) {
	c, err := aes.NewCipher(config.AuthSecret)
	if err != nil {
		return nil, fmt.Errorf("cannot create new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, fmt.Errorf("cannot create gcm from cipher: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("bad nonce size")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	rawUID, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot decode cyphertest: %w", err)
	}

	uid, err := uuid.FromBytes(rawUID)
	if err != nil {
		return nil, fmt.Errorf("cannot decode uid: %w", err)
	}
	return &uid, nil
}

// DecodeUIDFromHex decode uid from hex
func DecodeUIDFromHex(s string) (*uuid.UUID, error) {
	h, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("cannot decode hex string to bytes: %w", err)
	}
	return DecodeUID(h)
}
