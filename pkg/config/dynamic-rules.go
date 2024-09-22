package config

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

type DynamicRulesFile struct {
	StaticParam string `json:"static_param"`
	//Format            string `json:"format"`
	ChecksumIndexes   []int  `json:"checksum_indexes"`
	ChecksumConstants []int  `json:"checksum_constants"`
	ChecksumConstant  int64  `json:"checksum_constant"`
	AppToken          string `json:"app_token"`
	Prefix            string `json:"prefix"`
	Suffix            string `json:"suffix"`
	// RemoveHeaders []string `json:"remove_headers"`
	// ErrorCode int `json:"error_code"`
	// Message string `json:"message"`

	FormatString string `json:"-"`
	//Version      int    `json:"-"`
}

func (r *DynamicRulesFile) ParseFormatString() error {
	idx := strings.IndexRune(r.Format, ':')
	if idx < 2 {
		return fmt.Errorf("version not found in format string: %s", r.Format)
	}
	v, err := strconv.ParseInt(r.Format[0:idx-1], 10, 32)
	if err != nil {
		return fmt.Errorf("version not found in format string: %s", r.Format)
	}
	r.Version = int(v)

	re := regexp.MustCompile(`\{(\:[a-z])?\}`) */
	r.FormatString = r.Prefix + ":%s:%s:" + r.Suffix

	return nil
}

func LoadDynamicRules(parentCtx context.Context) error {
	log.Infof("🧮 Loading dynamic rules")

	ctx, cancel := context.WithTimeout(parentCtx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, Global.GetString(KeyDynamicRulesFile), nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	DynamicRules = &DynamicRulesFile{}
	err = json.NewDecoder(res.Body).Decode(DynamicRules)
	if err != nil {
		return err
	}

	err = DynamicRules.ParseFormatString()
	if err != nil {
		return err
	}

	return nil
}
