package main

import (
	"archive/zip"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/mischief/httpreader"
)

type Store interface {
	ReadFile(filename string) ([]byte, error)
	ReadDir(dirname string) ([]os.FileInfo, error)
	Stat(name string) (os.FileInfo, error)
	Close() error
}

type fileStore struct {
	base string
}

func (f *fileStore) ReadFile(filename string) ([]byte, error) {
	path := filepath.Join(f.base, filename)
	return ioutil.ReadFile(path)
}

func (f *fileStore) ReadDir(dirname string) ([]os.FileInfo, error) {
	path := filepath.Join(f.base, dirname)
	// resolve symlinks
	fis, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for i, fi := range fis {
		if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
			sympath := filepath.Join(path, fi.Name())
			newfi, err := os.Stat(sympath)
			if err != nil {
				log.Printf("broken symlink %q: %s", sympath, err)
				continue
			}

			fis[i] = newfi
		}
	}

	return fis, nil
}

func (f *fileStore) Stat(name string) (os.FileInfo, error) {
	path := filepath.Join(f.base, name)
	return os.Stat(path)
}

func (f *fileStore) Close() error {
	return nil
}

type zipStore struct {
	archive *zip.Reader
	closer  func() error
	files   map[string]*zip.File
	dirs    map[string][]*zip.File
}

func openZipStore(file string) (*zipStore, error) {
	url, err := url.Parse(file)
	if err != nil {
		return nil, err
	}

	zs := &zipStore{
		files: make(map[string]*zip.File),
		dirs:  make(map[string][]*zip.File),
	}

	switch url.Scheme {
	case "", "file":
		r, err := zip.OpenReader(file)
		if err != nil {
			return nil, err
		}
		zs.closer = r.Close
		zs.archive = &r.Reader
	case "http", "https":
		ra, err := httpreader.NewReader(file)
		if err != nil {
			return nil, err
		}
		sz, err := ra.Size()
		if err != nil {
			return nil, err
		}
		r, err := zip.NewReader(ra, sz)
		if err != nil {
			return nil, err
		}
		zs.archive = r
	}

	log.Printf("zip %q loading files...", file)
	for _, file := range zs.archive.File {
		zs.files[file.Name] = file
		log.Printf("%v...", file.Name)
	}

	log.Printf("zip %q loading directories...", file)
	for _, dir := range zs.archive.File {
		if !strings.HasSuffix(dir.Name, "/") {
			continue
		}

		var fi []*zip.File
		for _, file := range zs.files {
			name := file.Name
			if name == dir.Name {
				continue
			}

			if strings.HasSuffix(name, "/") {
				name = name[:len(name)-1]
			}

			d, _ := filepath.Split(name)

			if d != dir.Name {
				continue
			}

			fi = append(fi, file)

			if strings.HasPrefix(file.Name, dir.Name) {
				log.Printf("%v in dir %v", file.Name, dir.Name)
			}
		}

		zs.dirs[dir.Name] = fi
		log.Printf("%v...", dir.Name)
	}

	return zs, nil
}

func (z *zipStore) ReadFile(filename string) ([]byte, error) {
	log.Printf("read of %v", filename)
	zf, ok := z.files[filename]
	if !ok {
		return nil, os.ErrNotExist
	}

	zr, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	zc, err := ioutil.ReadAll(zr)
	if err != nil {
		return nil, err
	}

	return zc, nil
}

func (z *zipStore) ReadDir(dirname string) ([]os.FileInfo, error) {
	var fi []os.FileInfo

	dir, ok := z.dirs[dirname]
	if !ok {
		return nil, os.ErrNotExist
	}

	for _, d := range dir {
		fi = append(fi, d.FileInfo())
	}

	return fi, nil
}

func (z *zipStore) Stat(name string) (os.FileInfo, error) {
	log.Printf("stat %v", name)
	if f, ok := z.files[name]; ok {
		return f.FileInfo(), nil
	}
	return nil, os.ErrNotExist
}

func (z *zipStore) Close() error {
	if z.closer != nil {
		return z.closer()
	}

	return nil
}
