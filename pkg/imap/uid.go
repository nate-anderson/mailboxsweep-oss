package imap

import "strconv"

type UidHashFunc func(uid string) (uint64, error)

func ImapUIDHash(uid string) (uint64, error) {
	u, err := strconv.ParseUint(uid, 10, 32)
	return u, err
}
