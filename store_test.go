package main

import (
	"testing"
)

func TestZipStore(t *testing.T) {
	file := "fixtures/site.zip"
	zs, err := openZipStore(file)
	if err != nil {
		t.Error(err)
		return
	}

	ls, err := zs.ReadDir("site/foo/")
	if err != nil {
		t.Error(err)
		return
	}

	t.Logf("file %v", ls[0].Name())
}
