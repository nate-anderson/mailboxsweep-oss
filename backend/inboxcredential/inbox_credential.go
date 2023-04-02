package inboxcredential

import (
	"context"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mailboxsweep"
	"time"
)

type Credential interface {
	Address() string
	Created() time.Time
	Expires() time.Time
	// encrypt in place
	encrypt(encryptor cipher.AEAD) error
	// decrypt in place
	decrypt(decryptor cipher.AEAD) error
}

// IMAPInboxCredential represents the user-provided login information for an inbox
type IMAPInboxCredential struct {
	EmailAddress string `dynamodbav:"address"`
	// base64-encoded byte array
	Nonce string `dynamodbav:"nonce"`

	nonceBytes []byte
	// base64-encoded ciphertext
	Password  string    `dynamodbav:"password"`
	CreatedAt time.Time `dynamodbav:"created"`
	ExpiresAt int64     `dynamodbav:"expires_at"`
	Encrypted bool      `dynamodbav:"encrypted"`
	IsIMAP    bool      `dynamodbav:"is_imap,omitempty"`
}

func NewIMAPInboxCredential(address, password string, nonceSize int) (*IMAPInboxCredential, error) {
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	encodedNonce := base64.RawStdEncoding.EncodeToString(nonce)
	now := time.Now()
	return &IMAPInboxCredential{
		EmailAddress: address,
		Password:     password,
		Nonce:        encodedNonce,
		nonceBytes:   nonce,
		CreatedAt:    now,
		ExpiresAt:    now.Add(mailboxsweep.DynamoUserDataTTL).Unix(),
		IsIMAP:       true,
	}, nil
}

func (i *IMAPInboxCredential) encrypt(encryptor cipher.AEAD) error {
	if i.Encrypted {
		return errors.New("cannot encrypt credential: already encrypted")
	}
	var b []byte
	b = encryptor.Seal(b, i.nonceBytes, []byte(i.Password), nil)
	i.Password = base64.RawStdEncoding.EncodeToString(b)
	i.Encrypted = true
	return nil
}

func (i *IMAPInboxCredential) decrypt(decryptor cipher.AEAD) error {
	if !i.Encrypted {
		return errors.New("cannot decrypt credential: already decrypted")
	}
	decodedNonce, err := base64.RawStdEncoding.DecodeString(i.Nonce)
	if err != nil {
		return fmt.Errorf("failed to decode nonce: %w", err)
	}

	decodedPasswordCipher, err := base64.RawStdEncoding.DecodeString(i.Password)
	if err != nil {
		return fmt.Errorf("failed to decode password ciphertext: %w", err)
	}

	var b []byte
	b, err = decryptor.Open(b, decodedNonce, decodedPasswordCipher, nil)
	if err != nil {
		return fmt.Errorf("failed to decrypt credential: %w", err)
	}

	i.Password = string(b)
	i.Encrypted = false

	return nil
}

func (i IMAPInboxCredential) Address() string {
	return i.EmailAddress
}

func (i IMAPInboxCredential) Created() time.Time {
	return i.CreatedAt
}

func (i IMAPInboxCredential) Expires() time.Time {
	return time.Unix(i.ExpiresAt, 0)
}

type InboxCredentialRepository interface {
	Upsert(ctx context.Context, credential Credential) error
	FindByAddress(ctx context.Context, address string) (Credential, error)
	DeleteByAddress(ctx context.Context, address string) error
}
