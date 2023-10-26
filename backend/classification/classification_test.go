package classification

import (
	"mailboxsweep/assert"
	"mailboxsweep/inbox"
	"testing"
)

func TestStripMessages(t *testing.T) {
	c := NewClassification("none", "level 1", "LEVEL_1")
	c.Messages = append(c.Messages, inbox.Message{UID: "1"})

	c1 := NewClassification("none", "level 2 left", "LEVEL_2")
	c1.Messages = append(c1.Messages, inbox.Message{UID: "2"})

	c2 := NewClassification("none", "level 2 right", "LEVEL_2")
	c2.Messages = append(c2.Messages, inbox.Message{UID: "3"})

	c.Children["left"] = c1
	c.Children["right"] = c2

	summary := c.withoutMessages()

	if len(summary.Messages) > 0 {
		t.Errorf("root node should have zero messages, has %+v", summary.Messages)
	}

	if len(summary.Children["left"].Messages) > 0 {
		t.Errorf("left child should have zero messages, has %+v", c1.Messages)
	}

	if len(summary.Children["right"].Messages) > 0 {
		t.Errorf("right child should have zero messages, has %+v", c2.Messages)
	}
}

func TestLeafClassifications(t *testing.T) {
	c := NewClassification("none", "level 1", "LEVEL_1")
	c.Messages = append(c.Messages, inbox.Message{UID: "1"})

	c1 := NewClassification("none", "level 2", "LEVEL_2")

	c2 := NewClassification("none", "level 3", "LEVEL_3")

	c3 := NewClassification("none", "level 4 left", "LEVEL_4")
	c3.Messages = append(c3.Messages, inbox.Message{UID: "2"})

	c4 := NewClassification("none", "level 4 right", "LEVEL_4")
	c4.Messages = append(c4.Messages, inbox.Message{UID: "3"})

	c.Children["child1"] = c1
	c1.Children["child2"] = c2
	c2.Children["left"] = c3
	c2.Children["right"] = c4

	leaves := c.Leaves()

	if len(leaves) != 2 {
		t.Errorf("should have 2 leaf nodes, got %d", len(leaves))
	}

	if uid := leaves["/child1/child2/left"].Messages[0].UID; uid != "2" {
		t.Errorf("expected a message with ID message2: got %s", uid)
	}

	if uid := leaves["/child1/child2/right"].Messages[0].UID; uid != "3" {
		t.Errorf("expected a message with ID message3: got %s", uid)
	}
}

func TestClassificationTraverse(t *testing.T) {
	c1 := NewClassification("none", "1", "LEVEL_1")
	c2 := NewClassification("none", "2", "LEVEL_2")
	c3 := NewClassification("target", "3", "LEVEL_3")
	c3.Count = 10

	c2.Children["3"] = c3
	c1.Children["2"] = c2

	node, ok := c1.Traverse([]string{"2", "3"})
	assert.
		That(t, ok).
		Equals(true)

	assert.
		That(t, node.Label).
		Equals(c3.Label)

	assert.
		That(t, node.Dimension).
		Equals(c3.Dimension)

	assert.
		That(t, node.TypeKey).
		Equals(c3.TypeKey)

}
