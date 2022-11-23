package mailboxsweep

type MessageID interface {
	uint32 | string
}

type DeletionSet[messageIDType MessageID] struct {
	Inbox string
	UIDs  []messageIDType
}

func (d *DeletionSet[t]) AddUID(uid t) {
	d.UIDs = append(d.UIDs, uid)
}
