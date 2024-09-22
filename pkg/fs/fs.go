package fs

import (
	"context"
	"io"
	"strings"
)

var SharedFS FS

type FS interface {
	Init(ctx context.Context) error
	Stat(ctx context.Context, path string) (int64, error)
	Get(ctx context.Context, name string, start int64, len int64, out io.Writer) (bool, error)
	Put(ctx context.Context, name string, in io.Reader) error
	Rename(ctx context.Context, oldName string, newName string) error
	Delete(ctx context.Context, name string) error
}

func InitShared(ctx context.Context) error {
	switch strings.ToLower(config.Global.GetString(config.KeyStorageType)) {
	case "s3", "minio":
		SharedFS = &S3{}
	case "local", "fs":
		SharedFS = &Local{}
	default:
		SharedFS = &Local{}
	}

	return SharedFS.Init(ctx)
}
