package types

import (
	"fmt"
	"io"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/spf13/cast"
)

type Sort int8

const (
	SortUnknown Sort = 0
	SortAsc     Sort = 1
	SortDesc    Sort = -1
)

func ReverseSort(sd Sort) Sort {
	return sd * -1
}

func MarshalSort(i Sort) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		switch i {
		case SortUnknown:
			_, _ = io.WriteString(w, `"unknown"`)
		case SortAsc:
			_, _ = io.WriteString(w, `"asc"`)
		case SortDesc:
			_, _ = io.WriteString(w, `"desc"`)
		default:
			_, _ = io.WriteString(w, `"unknown"`)
		}
	})
}

func UnmarshalSort(v any) (res Sort, err error) {
	s, err := cast.ToStringE(v)
	if err != nil {
		return 0, fmt.Errorf("cannot cast to string")
	}

	switch strings.ToLower(s) {
	case "1", "+1", "ascending", "asc":
		res = SortAsc
	case "-1", "descending", "des", "desc":
		res = SortDesc
	default:
		res = SortUnknown
	}

	return res, nil
}
