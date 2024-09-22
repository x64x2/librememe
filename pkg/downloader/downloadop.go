package downloader

import (
	"net/http"
)

type DownloadOp struct {
	Destination string
	Request     *http.Request
	OnComplete  func(hash []byte) error
}

func NewDownload(destination string, request *http.Request) *Download {
	return &Download{
		Destination: destination,
		Request:     request,
	}
}
