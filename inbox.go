package mailsweeper

// Inbox is an authenticated mail account
type Inbox[messageIDType MessageID] interface {
	Name() string
	Folders() ([]string, error)
	Folder(name string) (Folder[messageIDType], error)
	ValidityKeys() (map[string]uint32, error)
}
