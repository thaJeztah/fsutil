package fsutil

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestValidHardlinks(t *testing.T) {
	err := checkHardlinks(changeStream([]string{
		"ADD foo file",
		"ADD foo2 file >foo",
	}))
	assert.NilError(t, err)
}

func TestInvalideHardlinks(t *testing.T) {
	err := checkHardlinks(changeStream([]string{
		"ADD foo file >foo2",
		"ADD foo2 file",
	}))
	assert.Assert(t, err != nil)
}

func TestInvalideHardlinks2(t *testing.T) {
	err := checkHardlinks(changeStream([]string{
		"ADD foo file",
		"ADD foo2 file >bar",
	}))
	assert.Assert(t, err != nil)
}

func TestHardlinkToDir(t *testing.T) {
	err := checkHardlinks(changeStream([]string{
		"ADD foo dir",
		"ADD foo2 file >foo",
	}))
	assert.Assert(t, err != nil)
}

func TestHardlinkToSymlink(t *testing.T) {
	err := checkHardlinks(changeStream([]string{
		"ADD foo symlink /",
		"ADD foo2 file >foo",
	}))
	assert.Assert(t, err != nil)
}

func checkHardlinks(inp []*change) error {
	h := &Hardlinks{}
	for _, c := range inp {
		if err := h.HandleChange(c.kind, c.path, c.fi, nil); err != nil {
			return err
		}
	}
	return nil
}
