package exhentai

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

)

func GenSignHeader(reqUrl string, t *time.Time, userId string) (string, error) {
	u, err := url.Parse(reqUrl)
	if err != nil {
		return "", err
	}

	path := u.Path
	if u.RawQuery != "" {
		path += "?" + u.RawQuery
	}

	parts := []string{
		config.DynamicRules.StaticParam,
		strconv.FormatInt(t.UnixMilli(), 10),
		path,
		userId,
	}
	base := strings.Join(parts, "\n")
	hB := sha1.Sum([]byte(base))
	h := hex.EncodeToString(hB[:])

	chk := config.DynamicRules.ChecksumConstant
	for _, i := range config.DynamicRules.ChecksumIndexes {
		if i > len(h) {
			return "", fmt.Errorf("index out of range: %d", i)
		}
		chk += int64(h[i])
	}
	if chk < 0 {
		chk = chk * (-1)
	}
	chkEnc := strconv.FormatInt(chk, 16)

	return fmt.Sprintf(config.DynamicRules.FormatString, h, chkEnc), nil
}
