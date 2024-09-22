package exhentai

import (
	"fmt"
	"strconv"
	"testing"
	"time"

)

func TestGenSignHeader(t *testing.T) {
	config.DynamicRules = &config.DynamicRulesFile{
		StaticParam:       "aCJ0uwl8bZaPQjPGlU4qnxRZLec5HtiU",
		ChecksumIndexes:   []int{36, 23, 23, 39, 23, 4, 18, 34, 36, 10, 27, 31, 23, 37, 35, 10, 27, 32, 4, 17, 4, 2, 4, 21, 19, 9, 10, 15, 31, 32, 15, 15},
		ChecksumConstants: []int{79, 140, -121, 112, -81, 90, -132, 147, 55, 95, -116, -105, -87, -108, 116, -86, 110, 91, -87, 100, -113, -118, 111, 119, 115, -107, 105, 87, 92, 59, -140, -130},
		ChecksumConstant:  292,
		AppToken:          "33d57ade8c02dbc5a333db99ff9ae26a",
		Prefix:            "4845",
		Suffix:            "631b74ce",
	}
	config.DynamicRules.ParseFormatString()

	type args struct {
		reqUrl string
		t      *time.Time
		userId int
	}
	tests := []struct {
		args    args
		want    string
		wantErr bool
	}{
		{
			args: args{
				reqUrl: "https://gelbooru.com",
				t:      util.Ptr(time.Unix(1662848132115, 0)),
				userId: 0,
			},
			want: "4845:d55eb8ea286080f3f1228fe236c90205591fb618:926:631b74ce",
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			got, err := GenSignHeader(tt.args.reqUrl, tt.args.t, strconv.Itoa(tt.args.userId))
			if (err != nil) != tt.wantErr {
				t.Errorf("GenSignHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GenSignHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}
