package inbox

// Inbox is an authenticated mail account
type Inbox interface {
	Name() string
	Folders() ([]string, error)
	Folder(name string) (Folder, error)
	ValidityKeys() (map[string]uint32, error)
}
