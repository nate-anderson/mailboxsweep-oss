package mailsweeper

import "testing"

func TestStripMessages(t *testing.T) {
	c := NewClassification("none", "level 1")
	c.Messages = append(c.Messages, Message{ID: "message1"})

	c1 := NewClassification("none", "level 2 left")
	c1.Messages = append(c.Messages, Message{ID: "message2"})

	c2 := NewClassification("none", "level 2 right")
	c2.Messages = append(c2.Messages, Message{ID: "message3"})

	c.Children.Set("left", c1)
	c.Children.Set("right", c2)

	stripMessages(c)

	if len(c.Messages) > 0 {
		t.Errorf("root node should have zero messages, has %+v", c.Messages)
	}

	if len(c1.Messages) > 0 {
		t.Errorf("left child should have zero messages, has %+v", c1.Messages)
	}

	if len(c2.Messages) > 0 {
		t.Errorf("right child should have zero messages, has %+v", c2.Messages)
	}
}
