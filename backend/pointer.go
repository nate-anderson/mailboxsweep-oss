package mailboxsweep

func ptr[t any](val t) *t {
	return &val
}
