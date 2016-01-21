package main

import (
	"io/ioutil"
	"net/http"
	"os"
)

func readfile(fs http.FileSystem, path string) ([]byte, error) {
	f, err := fs.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	return ioutil.ReadAll(f)
}

func readdir(fs http.FileSystem, path string) ([]os.FileInfo, error) {
	f, err := fs.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	return f.Readdir(-1)
}
