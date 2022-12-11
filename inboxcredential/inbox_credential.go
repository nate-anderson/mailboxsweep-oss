package inboxcredential

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mailboxsweep"
	"time"
)

// InboxCredential represents the user-provided login information for an inbox
type InboxCredential struct {
	Address string `dynamodbav:"address"`
	// base64-encoded byte array
	Nonce *string `dynamodbav:"nonce"`
	// base64-encoded ciphertext
	Password string    `dynamodbav:"password"`
	Created  time.Time `dynamodbav:"created"`
	Expires  int64     `dynamodbav:"expires_at"`
}

func NewInboxCredential(address, password string, cipherKey []byte) (*InboxCredential, error) {
	block, err := aes.NewCipher(cipherKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	encryptor, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher AEAD: %w", err)
	}

	nonce := make([]byte, encryptor.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	var b []byte
	b = encryptor.Seal(b, nonce, []byte(password), nil)
	encodedNonce := base64.RawStdEncoding.EncodeToString(nonce)
	now := time.Now()
	return &InboxCredential{
		Address:  address,
		Password: base64.RawStdEncoding.EncodeToString(b),
		Nonce:    &encodedNonce,
		Created:  now,
		Expires:  now.Add(mailboxsweep.DynamoUserDataTTL).Unix(),
	}, nil
}

func (i InboxCredential) decryptPassword(cipherKey []byte) (string, error) {
	if i.Nonce == nil {
		return i.Password, nil
	}

	block, err := aes.NewCipher(cipherKey)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher (decrypt): %w", err)
	}

	encryptor, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher AEAD (decrypt): %w", err)
	}

	decodedNonce, err := base64.RawStdEncoding.DecodeString(*i.Nonce)
	if err != nil {
		return "", fmt.Errorf("failed to decode nonce: %w", err)
	}

	decodedPasswordCipher, err := base64.RawStdEncoding.DecodeString(i.Password)
	if err != nil {
		return "", fmt.Errorf("failed to decode password ciphertext: %w", err)
	}

	var b []byte
	b, err = encryptor.Open(b, decodedNonce, decodedPasswordCipher, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt credential: %w", err)
	}

	return string(b), nil
}

type InboxCredentialRepository interface {
	Upsert(ctx context.Context, credential InboxCredential) error
	FindByAddress(ctx context.Context, address string) (*InboxCredential, error)
	DeleteByAddress(ctx context.Context, address string) error
}
