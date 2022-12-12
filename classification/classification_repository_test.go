package classification

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"mailboxsweep/inbox"
	"testing"
	"time"
)

func TestClassificationRepositoryImplementations(t *testing.T) {
	ctx := context.Background()

	s3Client, bucket, teardown := mailboxsweep.SetupTestBucket(ctx, t)
	defer teardown()
	s3repo := NewS3ClassificationRepository(s3Client, bucket)

	testSender := inbox.Sender{
		Address:  "admin@example.com",
		Username: "admin",
		Domain:   "example.com",
		Name:     "Administrator",
	}
	validityKeys := map[string]uint32{"INBOX": 1}
	address := "admin@example.com"

	fileRepo := NewFileClassificationRepository(t.TempDir())

	repositories := map[string]ClassificationRepository{
		"s3":   s3repo,
		"file": fileRepo,
	}

	for repoType, repo := range repositories {
		c := NewClassification("none1", "level 1")
		c.Count = 2

		c1 := NewClassification("none2", "level 2 left")
		sendDate := time.Now()
		c1.Messages = append(c1.Messages, inbox.Message{
			ID:       "message2",
			Subject:  "left child subject",
			Sender:   testSender,
			SendDate: sendDate,
		})
		c1.Count = 1

		c2 := NewClassification("none3", "level 2 right")
		c2.Messages = append(c2.Messages, inbox.Message{ID: "message3"})
		c2.Count = 1

		c.Children["left"] = c1
		c.Children["right"] = c2

		t.Run("store", func(t *testing.T) {
			err := repo.Store(ctx, address, &c, validityKeys)
			assert.ThatError(t, err).IsNil()
		})

		t.Run("fetch summary", func(t *testing.T) {
			classification, err := repo.FetchSummary(ctx, address)
			assert.ThatError(t, err).IsNil()
			assert.
				That(t, classification).
				SoThat(repoType + " should fetch without error").
				NotEqual(nil)

			assert.
				That(t, classification.Classification.Label).
				SoThat(repoType + " should return correct top-level label").
				Equals("level 1")

			assert.
				That(t, classification.Classification.Dimension).
				SoThat(repoType + " should return correct top-level dimension").
				Equals("none1")
		})

		t.Run("fetch leaf", func(t *testing.T) {
			classification, err := repo.FetchLeaf(ctx, address, []string{"left"})
			assert.
				ThatError(t, err).
				SoThat(repoType + " returns leaf").
				IsNil()
			assert.
				That(t, classification.Label).
				SoThat(repoType + " fetches correct label").
				Equals("level 2 left")

			assert.
				That(t, classification.Dimension).
				SoThat(repoType + " fetches correct dimemnsion").
				Equals("none2")

			assert.
				ThatSlice(t, classification.Messages).
				SoThat(repoType + " fetches all leaf messages").
				HasLength(1)

			msg := classification.Messages[0]
			assert.
				That(t, msg.ID).
				SoThat(repoType + " message ID is correct").
				Equals("message2")

			assert.
				That(t, msg.Subject).
				SoThat(repoType + " message subject is correct").
				Equals("left child subject")

			assert.
				That(t, msg.Sender).
				SoThat(repoType + " message sender address is correct").
				Equals(testSender)
		})
	}
}
