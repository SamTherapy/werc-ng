package main

import (
	"testing"

	"github.com/coreos/fleet/log"
)

func TestZipStore(t *testing.T) {
	file := "fixtures/site.zip"
	zs, err := openZipStore(file)
	if err != nil {
		log.Error(err)
		return
	}

	ls, err := zs.ReadDir("site/foo/")
	if err != nil {
		log.Error(err)
		return
	}

	t.Logf("file %v", ls[0].Name())
}
