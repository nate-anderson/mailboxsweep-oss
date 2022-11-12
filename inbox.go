package mailsweeper

// Inbox is an authenticated mail account
type Inbox interface {
	Folders() ([]string, error)
	Folder(name string) (Folder, error)
}
