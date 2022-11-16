package mailsweeper

import "mailsweeper/pkg/syncmap"

// Classifications act as a tree. Only leaf nodes should have messages
type Classification struct {
	Dimension string                               `json:"dimension"`
	Label     string                               `json:"label"`
	Messages  []Message                            `json:"messages,omitempty"`
	Count     uint                                 `json:"count"`
	Children  syncmap.Map[string, *Classification] `json:"children,omitempty"`
}

func NewClassification(dimension, label string) *Classification {
	return &Classification{
		Dimension: dimension,
		Label:     label,
		Children:  syncmap.New[string, *Classification](),
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

// Store and retrieve classified messages
type ClassificationRepository interface {
	Store(inbox string, classification *Classification) error
	Fetch(inbox string, path []string, includeMessages bool) (root *Classification, err error)
}

// strip messages from the whole tree
func stripMessages(c *Classification) {
	if len(c.Messages) > 0 {
		c.Messages = nil
	}

	c.Children.RangeWrite(func(key string, value *Classification) {
		stripMessages(value)
	})
}
