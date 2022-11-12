package mailsweeper

type Inbox interface {
	Folders() ([]string, error)
	Folder(name string) (Folder, error)
}
