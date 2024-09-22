package types

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/spf13/cast"

	"codeberg.org/biggestfan24/myfans/pkg/db"
)

type Source int

const (
	SourceUnknown        = Source(0)
	Sourceexhentai       = Source(db.Sourceexhentai)
	SourceFansly         = Source(db.SourceFansly)
	Sourceexhentai = Source(db.Sourceexhentai)
	SourceexhentaiFansly   = Source(db.SourceexhentaiFansly)
)

func (s Source) IsValid() bool {
	switch s {
	case
		Sourceexhentai, SourceFansly,
		Sourceexhentai, SourceexhentaiFansly:
		return true
	default:
		return false
	}
}

func (s Source) String() string {
	switch s {
	case Sourceexhentai:
		return "exhentai"
	case SourceFansly:
		return "fansly"
	case Sourceexhentai:
		return "exhentai-exhentai"
	case SourceexhentaiFansly:
		return "exhentai-fansly"
	default:
		return "unknown"
	}
}

func MarshalSource(s Source) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, _ = io.WriteString(w, `"`+s.String()+`"`)
	})
}

func UnmarshalSource(v any) (res Source, err error) {
	s, err := cast.ToStringE(v)
	if err != nil {
		return 0, fmt.Errorf("cannot cast to string")
	}

	switch strings.ToLower(s) {
	case "exhentai", strconv.Itoa(int(Sourceexhentai)):
		res = Sourceexhentai
	case "fansly", strconv.Itoa(int(SourceFansly)):
		res = SourceFansly
	case "exhentai-exhentai", strconv.Itoa(int(Sourceexhentai)):
		res = Sourceexhentai
	case "exhentai-fansly", strconv.Itoa(int(SourceexhentaiFansly)):
		res = SourceexhentaiFansly
	default:
		res = SourceUnknown
	}

	return res, nil
}
