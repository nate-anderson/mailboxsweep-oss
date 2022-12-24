package classification

import "strings"

type Path []string

const sep = ","

func ParsePath(str string) Path {
	return Path(strings.Split(str, sep))
}

func (p Path) String() string {
	return strings.Join(p, sep)
}
