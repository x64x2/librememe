package util

import (
	"strings"
)

func RemoveQS(urlstr string) string {
	idx := strings.IndexRune(urlstr, '?')
	if idx < 1 {
		return urlstr
	}
	return urlstr[:idx]
}
