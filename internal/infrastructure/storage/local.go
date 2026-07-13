package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Local struct{ Dir, BaseURL string }

func (l Local) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) (string, error) {
	p := filepath.Join(l.Dir, filepath.FromSlash(key))
	if e := os.MkdirAll(filepath.Dir(p), 0750); e != nil {
		return "", e
	}
	f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0640)
	if e != nil {
		return "", e
	}
	defer f.Close()
	if _, e = io.Copy(f, r); e != nil {
		return "", e
	}
	return fmt.Sprintf("%s/media/%s", l.BaseURL, key), nil
}
func (l Local) Delete(_ context.Context, key string) error {
	return os.Remove(filepath.Join(l.Dir, filepath.FromSlash(key)))
}
