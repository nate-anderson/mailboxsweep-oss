package mailsweeper

type DeletionSet struct {
	Inbox string
	UIDs  []uint32
}

func (d *DeletionSet) AddUID(uid uint32) {
	d.UIDs = append(d.UIDs, uid)
}
