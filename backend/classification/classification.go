package classification

import (
	"context"
	"fmt"
	"mailboxsweep/inbox"
	"mailboxsweep/pkg/syncmap"
	"math"
	"path/filepath"
	"sort"
	"time"
)

// Classifications act as a tree. Only leaf nodes should have messages
// this is the DTO type used for writing files and API responses
type Classification struct {
	Dimension   string                    `json:"dimension"`
	TypeKey     string                    `json:"typeKey"`
	Label       string                    `json:"label"`
	Messages    []inbox.Message           `json:"messages,omitempty"`
	Count       uint                      `json:"count"`
	UnreadCount uint                      `json:"unreadCount"`
	Children    map[string]Classification `json:"children,omitempty"`
	// pagination
	RankIndex *int `json:"rank_index"`
}

func NewClassification(dimension, label, typeKey string) Classification {
	return Classification{
		TypeKey:   typeKey,
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
	ValidityKeys map[string]uint32 `json:"validity_keys"`
}

type ClassificationDocument struct {
	Meta           ClassificationMeta `json:"meta"`
	Classification *Classification    `json:"classification"`
	Page           int                `json:"page"`
	TotalPages     int                `json:"total_pages"`
}

// Store and retrieve classified messages
type ClassificationRepository interface {
	// store the whole tree at once
	Store(ctx context.Context, inbox string, classification *Classification, validityKeys map[string]uint32) error
	// fetch a page of the summary
	FetchSummary(ctx context.Context, inbox string, page int) (root *ClassificationDocument, err error)
	FetchLeaf(ctx context.Context, inbox string, path []string) (leaf *Classification, err error)
	GetLeavesOfPrefix(ctx context.Context, inbox string, prefix []string) ([]Path, error)
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
	Dimension   string
	TypeKey     string
	Label       string
	Messages    []inbox.Message
	Count       uint
	UnreadCount uint
	Children    syncmap.Map[string, *SafeClassification]
}

func NewSafeClassification(dimension, label, typeKey string) *SafeClassification {
	return &SafeClassification{
		Dimension: dimension,
		Label:     label,
		TypeKey:   typeKey,
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
		Dimension:   sc.Dimension,
		Label:       sc.Label,
		TypeKey:     sc.TypeKey,
		Messages:    sc.Messages,
		Count:       sc.Count,
		UnreadCount: sc.UnreadCount,
		Children:    dtoChildren,
	}
}

func sortAndPaginateChildren(root Classification, validityKeys map[string]uint32, pageSize int) []ClassificationDocument {
	n := len(root.Children)
	numPages := math.Ceil(
		float64(n) / float64(pageSize),
	)

	doc := ClassificationDocument{
		Meta: ClassificationMeta{
			Created:      time.Now(),
			ValidityKeys: validityKeys,
		},
		Classification: &Classification{
			Dimension: root.Dimension,
			TypeKey:   root.TypeKey,
			Label:     root.Label,
			Messages:  root.Messages,
			Count:     root.Count,
			RankIndex: root.RankIndex,
			Children:  make(map[string]Classification),
		},
		Page:       0,
		TotalPages: int(numPages),
	}

	results := []ClassificationDocument{}

	orderedKeys := make([]string, 0, len(root.Children))
	for key := range root.Children {
		orderedKeys = append(orderedKeys, key)
	}
	sort.Slice(orderedKeys, func(i, j int) bool {
		keyI := orderedKeys[i]
		keyJ := orderedKeys[j]
		return root.Children[keyI].Count > root.Children[keyJ].Count
	})

	i := 0
	for _, key := range orderedKeys {
		doc.Classification.Children[key] = root.Children[key]
		if len(doc.Classification.Children) >= pageSize || i == (n-1) {
			results = append(results, doc)
			doc.Page++
			// set this to a new value because it's a pointer
			doc.Classification = &Classification{
				Dimension: root.Dimension,
				TypeKey:   root.TypeKey,
				Label:     root.Label,
				Messages:  root.Messages,
				Count:     root.Count,
				RankIndex: root.RankIndex,
				Children:  make(map[string]Classification),
			}
		}
		i++
	}

	return results
}

func mergePaginatedDocs(paginated []ClassificationDocument) ClassificationDocument {
	if len(paginated) < 1 {
		return ClassificationDocument{}
	}

	joined := paginated[0]
	joined.Page = 0

	for _, p := range paginated {
		for key, value := range p.Classification.Children {
			joined.Classification.Children[key] = value
		}
	}

	return joined
}
