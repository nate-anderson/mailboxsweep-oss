package inbox

import (
	"context"
	"errors"
	"mailboxsweep"
	"mailboxsweep/assert"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestGmailProvider(t *testing.T) {
	ctx := context.Background()
	pr := NewGmailProvider(ctx, "testo", "testo", GmailAPIRateLimits{}, logrus.New())

	t.Run("wrap error", func(t *testing.T) {
		err := errors.New("bogus")
		wrapped := pr.WrapError(err)
		assert.ThatError(t, wrapped).Is(mailboxsweep.ProviderErrorGmail)
		assert.ThatError(t, wrapped).Is(err)

		var perr *mailboxsweep.ProviderError
		assert.That(t, errors.As(wrapped, &perr)).Equals(true)
		assert.That(t, perr.Provider).Equals("gmail")
	})
}
