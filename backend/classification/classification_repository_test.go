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
	s3repo := NewS3ClassificationRepository(s3Client, bucket, 2)

	testSender := inbox.Sender{
		Address:  "admin@example.com",
		Username: "admin",
		Domain:   "example.com",
		Name:     "Administrator",
	}
	validityKeys := map[string]uint32{"INBOX": 1}
	address := "admin@example.com"

	fileRepo := NewFileClassificationRepository(t.TempDir(), 2)

	repositories := map[string]ClassificationRepository{
		"s3":   s3repo,
		"file": fileRepo,
	}

	for repoType, repo := range repositories {
		c := NewClassification("none1", "level 1", "LEVEL_1")
		c.Count = 2

		c1 := NewClassification("none2", "level 2 left", "LEVEL_2")
		sendDate := time.Now()
		c1.Messages = append(c1.Messages, inbox.Message{
			UID:      2,
			Subject:  "left child subject",
			Sender:   testSender,
			SendDate: sendDate,
		})
		c1.Count = 1

		c2 := NewClassification("none3", "level 2 right", "LEVEL_2")
		c2.Messages = append(c2.Messages, inbox.Message{UID: 3})
		c2.Count = 1

		c.Children["left"] = c1
		c.Children["right"] = c2

		t.Run("store", func(t *testing.T) {
			err := repo.Store(ctx, address, &c, validityKeys)
			assert.ThatError(t, err).IsNil()
		})

		t.Run("fetch summary", func(t *testing.T) {
			classification, err := repo.FetchSummary(ctx, address, 0)
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

			assert.
				That(t, classification.Classification.TypeKey).
				SoThat(repoType + " should return correct classification type key").
				Equals("LEVEL_1")
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

			assert.
				That(t, classification.TypeKey).
				SoThat(repoType + " returns correct type key").
				Equals("LEVEL_2")

			msg := classification.Messages[0]
			assert.
				That(t, msg.UID).
				SoThat(repoType + " message ID is correct").
				Equals(2)

			assert.
				That(t, msg.Subject).
				SoThat(repoType + " message subject is correct").
				Equals("left child subject")

			assert.
				That(t, msg.Sender).
				SoThat(repoType + " message sender address is correct").
				Equals(testSender)
		})

		t.Run("get leaves of path", func(t *testing.T) {
			inbox := "testinboxbarebones@admin.net"
			cl := Classification{
				Label: "all",
				Children: map[string]Classification{
					"1L": {
						Children: map[string]Classification{
							"2LL": {
								Children: map[string]Classification{
									"3LLL": {},
									"3LLR": {},
								},
							},
							"2LR": {
								Children: map[string]Classification{
									"3LRL": {},
									"3LRR": {},
								},
							},
						},
					},
					"1R": {
						Children: map[string]Classification{
							"2RL": {
								Children: map[string]Classification{
									"3RLL": {},
									"3RLR": {},
								},
							},
							"2RR": {
								Children: map[string]Classification{
									"3RRL": {},
									"3RRR": {},
								},
							},
						},
					},
				},
			}
			err := repo.Store(ctx, inbox, &cl, map[string]uint32{"INBOX": 19084})
			assert.ThatError(t, err).IsNil()

			leaves, err := repo.GetLeavesOfPrefix(ctx, inbox, []string{"1L", "2LL"})
			assert.ThatError(t, err).SoThat(repoType + " should find leaves for known prefix").IsNil()

			assert.That(t, len(leaves)).Equals(2)
			pathStrings := make([]string, 2)
			for i, l := range leaves {
				pathStrings[i] = l.String()
			}

			leftPath := Path([]string{"1L", "2LL", "3LLL"})
			rightPath := Path([]string{"1L", "2LL", "3LLR"})

			assert.ThatSlice(t, pathStrings).SoThat(repoType + " should return all leaf paths").Contains(leftPath.String())
			assert.ThatSlice(t, pathStrings).SoThat(repoType + " should return all leaf paths").Contains(rightPath.String())
		})
	}
}
