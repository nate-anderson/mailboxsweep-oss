package inboxcredential

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mailboxsweep"
	"time"

	"golang.org/x/oauth2"
)

type GmailInboxCredential struct {
	EmailAddress string    `dynamodbav:"address"`
	Nonce        string    `dynamodbav:"nonce"`
	APIToken     string    `dynamodbav:"api_token"`
	RefreshToken string    `dynamodbav:"refresh_token"`
	TokenExpiry  time.Time `dynamodbav:"token_expiry"`

	CreatedAt time.Time `dynamodbav:"created"`
	// Dynamo TTL - different from token expiry because we may need to refresh
	ExpiresAt int64 `dynamodbav:"expires_at"`
	Encrypted bool  `dynamodbav:"encrypted"`
	IsGmail   bool  `dynamodbav:"is_gmail"`
}

func NewGmailInboxCredential(address, apiToken, refreshToken string, expiry time.Time) *GmailInboxCredential {
	now := time.Now()
	return &GmailInboxCredential{
		EmailAddress: address,
		APIToken:     apiToken,
		RefreshToken: refreshToken,
		CreatedAt:    now,
		TokenExpiry:  expiry,
		ExpiresAt:    now.Add(mailboxsweep.DynamoUserDataTTL).Unix(),
		IsGmail:      true,
	}
}

func (c GmailInboxCredential) Address() string {
	return c.EmailAddress
}

func (c GmailInboxCredential) Created() time.Time {
	return c.CreatedAt
}

func (c GmailInboxCredential) Expires() time.Time {
	return time.Unix(c.ExpiresAt, 0)
}

func (c *GmailInboxCredential) encrypt(encryptor cipher.AEAD) error {
	if c.Encrypted {
		return errors.New("cannot encrypt credential: already encrypted")
	}

	nonce := make([]byte, encryptor.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	c.Nonce = base64.RawStdEncoding.EncodeToString(nonce)

	var buf []byte
	buf = encryptor.Seal(buf, nonce, []byte(c.APIToken), nil)
	c.APIToken = base64.RawStdEncoding.EncodeToString(buf)

	buf = nil
	buf = encryptor.Seal(buf, nonce, []byte(c.RefreshToken), nil)
	c.RefreshToken = base64.RawStdEncoding.EncodeToString(buf)

	c.Encrypted = true
	return nil
}

func (c *GmailInboxCredential) decrypt(decryptor cipher.AEAD) error {
	if !c.Encrypted {
		return errors.New("cannot decrypt credential: already decrypted")
	}

	decodedNonce, err := base64.RawStdEncoding.DecodeString(c.Nonce)
	if err != nil {
		return fmt.Errorf("failed to decode nonce: %w", err)
	}

	decodedAPITokenCipher, err := base64.RawStdEncoding.DecodeString(c.APIToken)
	if err != nil {
		return fmt.Errorf("failed to decode password ciphertext: %w", err)
	}

	decodedRefreshTokenCipher, err := base64.RawStdEncoding.DecodeString(c.RefreshToken)
	if err != nil {
		return err
	}

	var buf []byte
	buf, err = decryptor.Open(buf, decodedNonce, decodedAPITokenCipher, nil)
	if err != nil {
		return err
	}
	c.APIToken = string(buf)

	buf = nil
	buf, err = decryptor.Open(buf, decodedNonce, decodedRefreshTokenCipher, nil)
	if err != nil {
		return err
	}
	c.RefreshToken = string(buf)

	return nil
}

func (c GmailInboxCredential) OAuth2Token() (*oauth2.Token, error) {
	if c.Encrypted {
		return nil, errors.New("cannot turn credential into oauth token without decrypting")
	}

	return &oauth2.Token{
		AccessToken:  c.APIToken,
		RefreshToken: c.RefreshToken,
		Expiry:       c.TokenExpiry,
	}, nil
}
