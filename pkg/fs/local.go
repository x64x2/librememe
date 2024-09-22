package fs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dolmen-go/contextio"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
)

type Local struct {
	storagePath string
}

func (f *Local) Init(ctx context.Context) error {
	p := config.Global.GetString(config.KeyStorageLocation)
	if p == "" {
		p = config.Global.GetString(config.KeyDataPath)
	}
	e, err := homedir.Expand(p)
	if err == nil {
		p = e
	}
	storagePath, err := filepath.Abs(p)
	if err != nil {
		return fmt.Errorf("storage location path %s is invalid", p)
	}

	stat, err := os.Stat(storagePath)
	if errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(storagePath, 0o775)
		if err != nil {
			return fmt.Errorf("failed to create storage location directory: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("Failed to stat storage location: %w", err)
	} else if !stat.IsDir() {
		return fmt.Errorf("storage location is not a directory")
	}

	f.storagePath = storagePath

	log.Infof("📂 Storage location: %s", storagePath)

	return nil
}

func (f *Local) Stat(ctx context.Context, path string) (int64, error) {
	s, err := os.Stat(filepath.Join(f.storagePath, path))
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
		return -1, err
	}
	return s.Size(), nil
}

func (f *Local) Get(ctx context.Context, name string, start int64, len int64, out io.Writer) (bool, error) {
	dest := filepath.Join(f.storagePath, name)

	file, err := os.Open(dest)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	if start > 0 {
		_, err = file.Seek(start, 0)
		if err != nil {
			return false, fmt.Errorf("failed to seek: %w", err)
		}
	}

	if len > 0 {
		_, err = io.CopyN(out, contextio.NewReader(ctx, file), len)
	} else {
		_, err = io.Copy(out, contextio.NewReader(ctx, file))
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	return true, nil
}

func (f *Local) Put(ctx context.Context, name string, in io.Reader) error {
	dir := filepath.Dir(name)
	if dir != "" {
		err := os.MkdirAll(filepath.Join(f.storagePath, dir), 0o775)
		if err != nil {
			return fmt.Errorf("failed to create folder: %w", err)
		}
	}

	dest := filepath.Join(f.storagePath, name)
	wipDest := dest + ".wip"
	file, err := os.Create(wipDest)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	_, err = io.Copy(file, contextio.NewReader(ctx, in))
	file.Close()
	if err != nil {
		rmErr := os.Remove(wipDest)
		if rmErr != nil {
			log.Warnf("Failed to remove temporary file '%s': %v", wipDest, rmErr)
		}
		return fmt.Errorf("failed to write to file: %w", err)
	}

	err = os.Rename(dest+".wip", dest)
	if err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

func (f *Local) Rename(ctx context.Context, oldName string, newName string) error {
	dir := filepath.Dir(newName)
	if dir != "" {
		err := os.MkdirAll(filepath.Join(f.storagePath, dir), 0o775)
		if err != nil {
			return fmt.Errorf("failed to create folder: %w", err)
		}
	}

	newName = filepath.Join(f.storagePath, newName)
	oldName = filepath.Join(f.storagePath, oldName)

	err := os.Rename(oldName, newName)
	if err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

func (f *Local) Delete(ctx context.Context, name string) error {
	name = filepath.Join(f.storagePath, name)
	err := os.Remove(name)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	deleteEmptyDirs(name)

	return nil
}

func deleteEmptyDirs(name string) {
	dir := filepath.Dir(name)
	list, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	if len(list) == 0 {
		err = os.Remove(dir)
		if err != nil {
			return
		}

		deleteEmptyDirs(dir)
	} else {
		remove := true
		for _, el := range list {
			if el.IsDir() {
				remove = false
				break
			}
			if !strings.HasPrefix(el.Name(), ".") && !strings.HasSuffix(el.Name(), ".wip") {
				remove = false
				break
			}
		}

		if remove {
			err = os.RemoveAll(dir)
			if err != nil {
				return
			}

			deleteEmptyDirs(dir)
		}
	}
}
