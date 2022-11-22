package mailsweeper

// Inbox is an authenticated mail account
type Inbox[messageIDType MessageID] interface {
	Folders() ([]string, error)
	Folder(name string) (Folder[messageIDType], error)
}
