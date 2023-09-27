package inbox

import (
	"errors"
	"mailboxsweep"
	"mailboxsweep/assert"
	"testing"
)

func TestIMAPProvider(t *testing.T) {
	pr := IMAPProvider{}

	t.Run("wrap error", func(t *testing.T) {
		err := errors.New("bogus")
		wrapped := pr.WrapError(err)
		assert.ThatError(t, wrapped).Is(mailboxsweep.ProviderErrorIMAP)
		assert.ThatError(t, wrapped).Is(err)

		var perr *mailboxsweep.ProviderError
		assert.That(t, errors.As(wrapped, &perr)).Equals(true)
		assert.That(t, perr.Provider).Equals("imap")
	})
}
