package main

import (
	"archive/zip"
	"io"
	"net/http"
	"os"
	"strings"

	"golang.org/x/tools/godoc/vfs/httpfs"
	"golang.org/x/tools/godoc/vfs/zipfs"
)

// Make a few FS.
// Supported types are:
// - local directory (foo/bar/)
// - local zip file (foo/bar.zip)
func NewFS(uri string) (*FS, error) {
	if strings.HasSuffix(uri, ".zip") {
		r, err := zip.OpenReader(uri)
		if err != nil {
			return nil, err
		}

		zfs := zipfs.New(r, uri)
		hfs := httpfs.New(zfs)
		fs := &FS{
			FileSystem: hfs,
			closeme:    r,
		}

		return fs, nil
	}

	fi, err := os.Stat(uri)
	if err != nil {
		return nil, err
	}

	if !fi.IsDir() {
		return nil, os.ErrInvalid
	}

	fs := &FS{
		FileSystem: http.Dir(uri),
	}

	return fs, nil
}

type FS struct {
	http.FileSystem
	closeme io.Closer
}

func (f *FS) Close() error {
	if f.closeme != nil {
		return f.Close()
	}
	return nil
}
