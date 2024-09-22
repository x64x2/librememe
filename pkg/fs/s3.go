package fs

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
)

type S3 struct {
	client *minio.Client
	bucket string
}

func (f *S3) Init(parentCtx context.Context) error {
	v := config.Global.GetString(config.KeyStorageLocation)
	u, err := url.Parse(v)
	if err != nil || u == nil || u.User == nil || u.Path == "" {
		return fmt.Errorf("invalid data path for S3: '%s'", v)
	}

	pass, _ := u.User.Password()
	f.client, err = minio.New(u.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(u.User.Username(), pass, ""),
		Secure: u.Scheme == "https",
		Region: u.Query().Get("region"),
	})
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	f.bucket = strings.TrimPrefix(u.Path, "/")

	ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	exists, err := f.client.BucketExists(ctx, f.bucket)
	cancel()
	if err != nil {
		return fmt.Errorf("failed to check for bucket: %w", err)
	}
	if !exists {
		ctx, cancel = context.WithTimeout(parentCtx, 30*time.Second)
		err = f.client.MakeBucket(ctx, f.bucket, minio.MakeBucketOptions{
			Region: u.Query().Get("region"),
		})
		cancel()
		if err != nil {
			return fmt.Errorf("bucket '%s' does not exist and could not be created", f.bucket)
		}
	}

	log.Infof("📂 Storing on S3 host: '%s' bucket: '%s'", u.Host, f.bucket)

	return nil
}

func (f *S3) Stat(parentCtx context.Context, path string) (int64, error) {
	ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	stat, err := f.client.StatObject(ctx, f.bucket, path, minio.StatObjectOptions{})
	cancel()
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			err = nil
		}
		return -1, err
	}
	return stat.Size, nil
}

func (f *S3) Get(ctx context.Context, name string, start int64, len int64, out io.Writer) (bool, error) {
	getOpts := minio.GetObjectOptions{}
	if start > 0 || len > 0 {
		var end int64 = 0
		if len > 0 {
			end = start + len - 1
		}
		getOpts.SetRange(start, end)
	}

	obj, err := f.client.GetObject(ctx, f.bucket, name, getOpts)
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}

	_, err = io.Copy(out, obj)
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (f *S3) Put(ctx context.Context, name string, in io.Reader) error {
	_, err := f.client.PutObject(ctx, f.bucket, name, in, -1, minio.PutObjectOptions{})
	return err
}

func (f *S3) Rename(ctx context.Context, oldName string, newName string) error {
	_, err := f.client.CopyObject(
		ctx,
		minio.CopyDestOptions{
			Bucket: f.bucket,
			Object: newName,
		},
		minio.CopySrcOptions{
			Bucket: f.bucket,
			Object: oldName,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	return f.Delete(ctx, oldName)
}

func (f *S3) Delete(parentCtx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	defer cancel()

	err := f.client.RemoveObject(ctx, f.bucket, name, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}
