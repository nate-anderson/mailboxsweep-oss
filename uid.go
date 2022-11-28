package mailboxsweep

import "strconv"

type uidHashFunc func(uid string) (uint64, error)

func imapUIDHash(uid string) (uint64, error) {
	u, err := strconv.ParseUint(uid, 10, 32)
	return u, err
}
