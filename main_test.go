package main

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/bradleyjkemp/cupaloy"
	"github.com/stretchr/testify/assert"
)

func TestSnapshot(t *testing.T) {
	f, err := os.Open(".snapshots/fluxlogs-source.ndjson")
	assert.NoError(t, err)
	defer f.Close()
	dst := bytes.NewBuffer(nil)
	w, err := io.Copy(dst, convert(f))
	assert.True(t, w > 0)
	assert.NoError(t, err)
	assert.Equal(t, w, int64(dst.Len()))
	c := cupaloy.New(cupaloy.SnapshotFileExtension(".ndjson"))
	if err := c.Snapshot(dst.String()); err != nil {
		t.Log(err.Error())
		t.Log("Snapshot failed. To update snapshot, run 'UPDATE_SNAPSHOTS=1 go test ./'")
		t.Fail()
	}
}
