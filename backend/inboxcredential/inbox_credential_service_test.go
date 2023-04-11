package inboxcredential

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"testing"
	"time"
)

func TestInboxCredentialService(t *testing.T) {
	cl := mailboxsweep.SetupTestDynamoClient()
	ctx := context.Background()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, cl)
	defer teardown()

	repo := NewDynamoInboxCredentialRepository(cl, table)
	cipherKey := []byte("655468576D5A7134")
	service, err := NewService(repo, cipherKey)
	assert.ThatError(t, err).IsNil()

	testAddr := "secure_friend@example.com"
	testPassword := "great_password_123"
	authedCtx := mailboxsweep.AuthorizeAPIContext(ctx, testAddr)

	t.Run("IMAP credentials", func(t *testing.T) {
		t.Run("store credential", func(t *testing.T) {
			cred := NewIMAPInboxCredential(testAddr, testPassword)

			t.Run("unauthenticated, failure", func(t *testing.T) {
				err := service.StoreCredential(ctx, cred)
				assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			})

			t.Run("success", func(t *testing.T) {
				err := service.StoreCredential(authedCtx, cred)
				assert.ThatError(t, err).IsNil()
			})
		})

		t.Run("retrieve credential", func(t *testing.T) {
			t.Run("unauthenticated, failure", func(t *testing.T) {
				ret, err := service.FindCredential(ctx)
				assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
				assert.That(t, ret == nil).Equals(true)
			})

			t.Run("success", func(t *testing.T) {
				ret, err := service.FindCredential(authedCtx)
				assert.ThatError(t, err).IsNil()

				imapCred, ok := ret.(*IMAPInboxCredential)
				assert.That(t, ok).Equals(true)

				assert.That(t, imapCred.Password).NotEqual("")
				t.Run("password is encrypted", func(t *testing.T) {
					assert.ThatString(t, imapCred.Password).NotContains(testPassword)
				})

				t.Run("decrypt password", func(t *testing.T) {
					t.Run("unauthenticated, faulre", func(t *testing.T) {
						decrypted, err := service.DecryptCredential(ctx, ret)
						assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
						assert.That(t, decrypted == nil).Equals(true)
					})

					t.Run("success", func(t *testing.T) {
						decrypted, err := service.DecryptCredential(authedCtx, ret)
						assert.ThatError(t, err).IsNil()
						imapCred, ok := decrypted.(*IMAPInboxCredential)
						assert.That(t, ok).Equals(true)
						assert.That(t, imapCred.Password).Equals(testPassword)
					})
				})
			})
		})

		t.Run("delete credential", func(t *testing.T) {
			t.Run("unauthenticated, failure", func(t *testing.T) {
				err := service.DeleteCredential(ctx)
				assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			})

			t.Run("success", func(t *testing.T) {
				err := service.DeleteCredential(authedCtx)
				assert.ThatError(t, err).IsNil()

				t.Run("record is not present in DB", func(t *testing.T) {
					ret, err := service.FindCredential(authedCtx)
					assert.ThatError(t, err).Is(mailboxsweep.ErrNoStoredCredential)
					assert.That(t, ret == nil).Equals(true)
				})
			})
		})
	})

	t.Run("Gmail credentials", func(t *testing.T) {
		apiToken := "test_token_123"
		refreshToken := "refresh_token_456"
		expiry := time.Now().Add(time.Hour)

		t.Run("store credential", func(t *testing.T) {
			cred := NewGmailInboxCredential(testAddr, apiToken, refreshToken, expiry)

			t.Run("unauthenticated, failure", func(t *testing.T) {
				err := service.StoreCredential(ctx, cred)
				assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			})

			t.Run("success", func(t *testing.T) {
				err := service.StoreCredential(authedCtx, cred)
				assert.ThatError(t, err).IsNil()
			})
		})

		t.Run("retrieve credential", func(t *testing.T) {
			t.Run("unauthenticated, failure", func(t *testing.T) {
				ret, err := service.FindCredential(ctx)
				assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
				assert.That(t, ret == nil).Equals(true)
			})

			t.Run("success", func(t *testing.T) {
				ret, err := service.FindCredential(authedCtx)
				assert.ThatError(t, err).IsNil()

				gmailCred, ok := ret.(*GmailInboxCredential)
				assert.That(t, ok).Equals(true)

				assert.That(t, gmailCred.APIToken).NotEqual("")
				assert.That(t, gmailCred.RefreshToken).NotEqual("")
				assert.That(t, gmailCred.TokenExpiry.IsZero()).Equals(false)

				t.Run("secure data is encrypted", func(t *testing.T) {
					assert.ThatString(t, gmailCred.APIToken).NotContains(testPassword)
					assert.ThatString(t, gmailCred.RefreshToken).NotContains(refreshToken)
				})

				t.Run("decrypt password", func(t *testing.T) {
					t.Run("unauthenticated, failure", func(t *testing.T) {
						decrypted, err := service.DecryptCredential(ctx, ret)
						assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
						assert.That(t, decrypted == nil).Equals(true)
					})

					t.Run("authenticated, success", func(t *testing.T) {
						decrypted, err := service.DecryptCredential(authedCtx, ret)
						assert.ThatError(t, err).IsNil()
						gmailCred, ok := decrypted.(*GmailInboxCredential)
						assert.That(t, ok).Equals(true)
						assert.That(t, gmailCred.APIToken).Equals(apiToken)
						assert.That(t, gmailCred.RefreshToken).Equals(refreshToken)
						assert.That(t, gmailCred.TokenExpiry.Equal(expiry)).Equals(true)
						assert.That(t, gmailCred.Encrypted).Equals(false)
					})
				})
			})
		})

		t.Run("delete credential", func(t *testing.T) {
			t.Run("unauthenticated, failure", func(t *testing.T) {
				err := service.DeleteCredential(ctx)
				assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			})

			t.Run("success", func(t *testing.T) {
				err := service.DeleteCredential(authedCtx)
				assert.ThatError(t, err).IsNil()

				t.Run("record is not present in DB", func(t *testing.T) {
					ret, err := service.FindCredential(authedCtx)
					assert.ThatError(t, err).Is(mailboxsweep.ErrNoStoredCredential)
					assert.That(t, ret == nil).Equals(true)
				})
			})
		})
	})

}
