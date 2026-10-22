package upload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

const (
	KindFile       = "file"
	KindPhoto      = "photo"
	KindAttachment = "attachment"
)

var ErrFileNotFound = errors.New("upload: no stored file has this name")

type Blob struct {
	Data    []byte
	ModTime time.Time
}

type Store interface {
	Put(ctx context.Context, kind, name, mime string, data []byte) error
	Get(ctx context.Context, kind, name string) (Blob, error)
	Delete(ctx context.Context, kind, name string) error
}

type Disk struct {
	root string
}

func NewDisk(root string) *Disk {
	if root == "" {
		root = defaultDir
	}
	return &Disk{root: root}
}

func (d *Disk) Root() string {
	return d.root
}

func (d *Disk) Dir(kind string) string {
	switch kind {
	case KindPhoto:
		return filepath.Join(d.root, PhotoDirName)
	case KindAttachment:
		return filepath.Join(d.root, AttachmentDirName)
	default:
		return d.root
	}
}

func (d *Disk) Put(ctx context.Context, kind, name, mime string, data []byte) error {
	dir := d.Dir(kind)
	if err := os.MkdirAll(dir, directoryMode); err != nil {
		return fmt.Errorf("upload: could not create the folder %s: %w", dir, err)
	}
	target := filepath.Join(dir, name)
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fileMode)
	if err != nil {
		return fmt.Errorf("upload: could not create the file %s: %w", name, err)
	}
	_, err = file.Write(data)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(target)
		return fmt.Errorf("upload: could not write the file %s: %w", name, err)
	}
	return nil
}

func (d *Disk) Get(ctx context.Context, kind, name string) (Blob, error) {
	file, err := os.Open(filepath.Join(d.Dir(kind), name))
	if err != nil {
		return Blob{}, ErrFileNotFound
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		return Blob{}, ErrFileNotFound
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return Blob{}, fmt.Errorf("upload: could not read the file %s: %w", name, err)
	}
	return Blob{Data: data, ModTime: info.ModTime()}, nil
}

func (d *Disk) Delete(ctx context.Context, kind, name string) error {
	err := os.Remove(filepath.Join(d.Dir(kind), name))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("upload: could not delete the file %s: %w", name, err)
	}
	return nil
}

func readLimited(file io.Reader, limit int64) ([]byte, bool, error) {
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, false, err
	}
	return data, int64(len(data)) > limit, nil
}

func StoredName(kind, name string) bool {
	switch kind {
	case KindPhoto:
		return photoNamePattern.MatchString(name)
	case KindAttachment:
		return attachmentNamePattern.MatchString(name)
	default:
		return namePattern.MatchString(name)
	}
}
