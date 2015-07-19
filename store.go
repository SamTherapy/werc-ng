package main

import (
	"archive/zip"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Store interface {
	ReadFile(filename string) ([]byte, error)
	ReadDir(dirname string) ([]os.FileInfo, error)
	Stat(name string) (os.FileInfo, error)
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
	return ioutil.ReadDir(path)
}

func (f *fileStore) Stat(name string) (os.FileInfo, error) {
	path := filepath.Join(f.base, name)
	return os.Stat(path)
}

type zipStore struct {
	archive *zip.ReadCloser
	files   map[string]*zip.File
	dirs    map[string][]*zip.File
}

func openZipStore(file string) (*zipStore, error) {
	r, err := zip.OpenReader(file)
	if err != nil {
		log.Fatal(err)
	}

	zs := &zipStore{r, make(map[string]*zip.File), make(map[string][]*zip.File)}

	log.Printf("zip %q loading files...", file)
	for _, file := range r.File {
		zs.files[file.Name] = file
		log.Printf("%v...", file.Name)
	}

	log.Printf("zip %q loading directories...", file)
	for _, dir := range r.File {
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
