package mailboxsweep

import (
	"context"
	"fmt"
	"mailboxsweep/pkg/syncmap"
	"path/filepath"
	"time"
)

// Classifications act as a tree. Only leaf nodes should have messages
// this is the DTO type used for writing files and API responses
type Classification struct {
	Dimension string                    `json:"dimension"`
	Label     string                    `json:"label"`
	Messages  []Message                 `json:"messages,omitempty"`
	Count     uint                      `json:"count"`
	Children  map[string]Classification `json:"children,omitempty"`
}

func NewClassification(dimension, label string) Classification {
	return Classification{
		Dimension: dimension,
		Label:     label,
		Children:  make(map[string]Classification),
	}
}

/*
	An example classification tree might look like this
	costco.com
	 |- noreply.ads@costco.com
	     |- Before 01/01/2020
		 	|- 586BBB6E-7D40-4054-8E77-3F927676F951@gmail.com
			|- 7c7240220809181746m4f2b606bqf2b1f0da18b89e2d@mail.gmail.com
			|- 4865787b520f59e79d2c0d6ebb0c3ffc@localhost.localdomain
		 |- After 01/01/2020
	 |- promotions@@costco.com
	 	 |- Before 01/01/2020
		 	|- 1878927831-1220919904-cardhu_decombobulator_blackberry.rim.net-1474969766-@bxe289.bisx.prod.on.blackberry
		 |- After 01/01/2020
	 |- receipts@costco.com
	 	 |- Before 01/01/2020
	gmail.com
	 |- donna@gmail.com
	 	 |- Before 01/01/2020
			|- 1F8E200A-44E5-4182-85F8-0A40CC799EB0@gmail.com
		 |- After 01/01/2020
	 |- liferuiner8@gmail.com
	 	 |- Before 01/01/2020
		 |- After 01/01/2020
	amazon.com
	 |- noreply@amazon.com
	 	 |- Before 01/01/2020
		 |- After 01/01/2020


	This allows increasing granularity as we mark messages to delete/mark read
		- Delete all from this domain?
		- Delete all from this sender?
		- Delete all from this sender sent before the specified date?
*/

type ClassificationMeta struct {
	Created time.Time `json:"created"`
	// store folder validity key maps to know when our stored UIDs are invalidated
	ValidityKeys map[string]uint32 `json:"validityKeys"`
}

type ClassificationDocument struct {
	Meta           ClassificationMeta `json:"meta"`
	Classification *Classification    `json:"classification"`
}

// Store and retrieve classified messages
type ClassificationRepository interface {
	Store(ctx context.Context, inbox string, classification *Classification, validityKeys map[string]uint32) error
	FetchSummary(ctx context.Context, inbox string) (root *ClassificationDocument, err error)
	FetchLeaf(ctx context.Context, inbox string, path []string) (leaf *Classification, err error)
}

// return a copy of c with the messages removed recursively
func (c Classification) withoutMessages() Classification {
	cop := c
	cop.Messages = nil
	if len(c.Children) == 0 {
		return cop
	}

	for key, value := range cop.Children {
		cop.Children[key] = value.withoutMessages()
	}

	return cop
}

func (c *Classification) Leaves() map[string]Classification {
	if len(c.Children) == 0 {
		// this is a leaf
		return map[string]Classification{"/": *c}
	}

	cls := map[string]Classification{}
	for key, cl := range c.Children {
		// recursively filter out non-leaves
		leaves := cl.Leaves()
		for childKey, childLeaf := range leaves {
			p := filepath.Join("/", key, childKey)
			cls[p] = childLeaf
		}
	}

	return cls
}

func (c Classification) Traverse(path []string) (Classification, bool) {
	node := c
	for _, key := range path {
		var ok bool
		node, ok = node.Children[key]
		if !ok {
			return Classification{}, false
		}
	}
	return node, true
}

func (c *Classification) DebugString() string {
	return fmt.Sprintf("LABEL: '%s' (%d count, %d messages, %d children, dimension '%s')", c.Label, c.Count, len(c.Messages), len(c.Children), c.Dimension)
}

func (c *Classification) TreeString() string {
	if len(c.Children) == 0 {
		return c.DebugString()
	}

	str := c.DebugString() + "\n"
	for key, value := range c.Children {
		str += fmt.Sprintf("    %s :: %s\n", key, value.TreeString())
	}

	return str
}

// Like a classification, but safe for concurrent writes
// this is needed internally because we receive IMAP messages over
// one or more goroutines, and need to safely update the classification
// tree
type SafeClassification struct {
	Dimension string
	Label     string
	Messages  []Message
	Count     uint
	Children  syncmap.Map[string, *SafeClassification]
}

func NewSafeClassification(dimension, label string) *SafeClassification {
	return &SafeClassification{
		Dimension: dimension,
		Label:     label,
		Children:  syncmap.New[string, *SafeClassification](),
	}
}

func (sc *SafeClassification) Classification() Classification {
	children := sc.Children.Map()
	dtoChildren := make(map[string]Classification)
	if len(children) > 0 {
		for key, value := range children {
			cl := value.Classification()
			dtoChildren[key] = cl
		}
	}

	return Classification{
		Dimension: sc.Dimension,
		Label:     sc.Label,
		Messages:  sc.Messages,
		Count:     sc.Count,
		Children:  dtoChildren,
	}
}
