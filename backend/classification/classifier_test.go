package classification

import (
	"fmt"
	"mailboxsweep/assert"
	"mailboxsweep/inbox"
	"strings"
	"testing"
)

func TestClassifiers(t *testing.T) {
	t.Run("subject keyword", func(t *testing.T) {
		keywords := []string{"bill", "10%", "5098", "me@me.com"}
		matchSubjectFormats := []string{"%s now", "hello %s you", "%s", "%s's baby"}
		notMatchSubjectFormats := []string{"%sflambe", "oija%s"}

		for _, keyword := range keywords {
			classifier, err := NewSubjectKeywordClassifier(keyword)
			assert.ThatError(t, err).IsNil()
			assert.ThatString(t, classifier.keyword).NotContains(" ")

			for i, matchFmt := range matchSubjectFormats {
				subj := fmt.Sprintf(matchFmt, keyword)
				msg := inbox.Message{
					UID:     uint32(i),
					Folder:  "bs-folder",
					Subject: subj,
				}
				key := classifier.Key(msg)
				assert.That(t, key).
					SoThat("matching subject format: " + matchFmt).
					Equals(fmt.Sprintf("contains_keyword_%s", strings.ToLower(strings.TrimSpace(keyword))))
				class := classifier.Classify(msg)
				assert.That(t, class).Equals(fmt.Sprintf("Subject includes \"%s\"", strings.ToLower(strings.TrimSpace(keyword))))
			}

			for i, noMatchFmt := range notMatchSubjectFormats {
				subj := fmt.Sprintf(noMatchFmt, keyword)
				msg := inbox.Message{
					UID:     uint32(i),
					Folder:  "bs-folder",
					Subject: subj,
				}
				key := classifier.Key(msg)
				assert.That(t, key).
					SoThat("non-matching subject format: " + noMatchFmt + " " + subj).
					Equals(fmt.Sprintf("not_contains_keyword_%s", strings.ToLower(strings.TrimSpace(keyword))))
				class := classifier.Classify(msg)
				assert.That(t, class).Equals(fmt.Sprintf("Subject does not include \"%s\"", strings.ToLower(strings.TrimSpace(keyword))))
			}
		}
	})
}
