package fsutil

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/containerd/continuity/fs/fstest"
	"gotest.tools/v3/assert"
	fsu "gotest.tools/v3/fs"
)

func TestFollowLinks(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "test")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpDir)

	apply := fstest.Apply(
		fstest.CreateDir("dir", 0700),
		fstest.CreateFile("dir/foo", []byte("contents"), 0600),
		fstest.Symlink("foo", "dir/l1"),
		fstest.Symlink("dir/l1", "l2"),
		fstest.CreateFile("bar", nil, 0600),
		fstest.CreateFile("baz", nil, 0600),
	)

	assert.NilError(t, apply.Apply(tmpDir))

	out, err := FollowLinks(tmpDir, []string{"l2", "bar"})
	assert.NilError(t, err)

	assert.Equal(t, out, []string{"bar", "dir/foo", "dir/l1", "l2"})
}

func TestFollowLinksLoop(t *testing.T) {
	tmpDir := fsu.NewDir(t, t.Name())
	defer tmpDir.Remove()

	apply := fstest.Apply(
		fstest.Symlink("l1", "l1"),
		fstest.Symlink("l2", "l3"),
		fstest.Symlink("l3", "l2"),
	)
	assert.NilError(t, apply.Apply(tmpDir.Path()))

	out, err := FollowLinks(tmpDir.Path(), []string{"l1", "l3"})
	assert.NilError(t, err)

	assert.Equal(t, out, []string{"l1", "l2", "l3"})
}

func TestFollowLinksAbsolute(t *testing.T) {
	tmpDir := fsu.NewDir(t, t.Name())
	defer tmpDir.Remove()

	apply := fstest.Apply(
		fstest.CreateDir("dir", 0700),
		fstest.Symlink("/foo/bar/baz", "dir/l1"),
		fstest.CreateDir("foo", 0700),
		fstest.Symlink("../", "foo/bar"),
		fstest.CreateFile("baz", nil, 0600),
	)
	assert.NilError(t, apply.Apply(tmpDir.Path()))

	out, err := FollowLinks(tmpDir.Path(), []string{"dir/l1"})
	assert.NilError(t, err)

	assert.Equal(t, out, []string{"baz", "dir/l1", "foo/bar"})

	// same but a link outside root
	tmpDir2 := fsu.NewDir(t, t.Name())
	defer tmpDir2.Remove()

	apply = fstest.Apply(
		fstest.CreateDir("dir", 0700),
		fstest.Symlink("/foo/bar/baz", "dir/l1"),
		fstest.CreateDir("foo", 0700),
		fstest.Symlink("../../../", "foo/bar"),
		fstest.CreateFile("baz", nil, 0600),
	)
	assert.NilError(t, apply.Apply(tmpDir2.Path()))

	out, err = FollowLinks(tmpDir2.Path(), []string{"dir/l1"})
	assert.NilError(t, err)

	assert.Equal(t, out, []string{"baz", "dir/l1", "foo/bar"})
}

func TestFollowLinksNotExists(t *testing.T) {
	tmpDir := fsu.NewDir(t, t.Name())
	defer tmpDir.Remove()

	out, err := FollowLinks(tmpDir.Path(), []string{"foo/bar/baz", "bar/baz"})
	assert.NilError(t, err)

	assert.Equal(t, out, []string{"bar/baz", "foo/bar/baz"})

	// root works fine with empty directory
	out, err = FollowLinks(tmpDir.Path(), []string{"."})
	assert.NilError(t, err)

	assert.Equal(t, out, []string(nil))

	out, err = FollowLinks(tmpDir.Path(), []string{"f*/foo/t*"})
	assert.NilError(t, err)

	assert.Equal(t, out, []string{"f*/foo/t*"})
}

func TestFollowLinksNormalized(t *testing.T) {
	tmpDir := fsu.NewDir(t, t.Name())
	defer tmpDir.Remove()

	out, err := FollowLinks(tmpDir.Path(), []string{"foo/bar/baz", "foo/bar"})
	assert.NilError(t, err)

	assert.Equal(t, out, []string{"foo/bar"})

	apply := fstest.Apply(
		fstest.CreateDir("dir", 0700),
		fstest.Symlink("/foo", "dir/l1"),
		fstest.Symlink("/", "dir/l2"),
		fstest.CreateDir("foo", 0700),
		fstest.CreateFile("foo/bar", nil, 0600),
	)
	assert.NilError(t, apply.Apply(tmpDir.Path()))

	out, err = FollowLinks(tmpDir.Path(), []string{"dir/l1", "foo/bar"})
	assert.NilError(t, err)

	assert.Equal(t, out, []string{"dir/l1", "foo"})

	out, err = FollowLinks(tmpDir.Path(), []string{"dir/l2", "foo", "foo/bar"})
	assert.NilError(t, err)

	assert.Equal(t, out, []string(nil))
}

func TestFollowLinksWildcard(t *testing.T) {
	tmpDir := fsu.NewDir(t, t.Name())
	defer tmpDir.Remove()

	apply := fstest.Apply(
		fstest.CreateDir("dir", 0700),
		fstest.CreateDir("foo", 0700),
		fstest.Symlink("/foo/bar1", "dir/l1"),
		fstest.Symlink("/foo/bar2", "dir/l2"),
		fstest.Symlink("/foo/bar3", "dir/anotherlink"),
		fstest.Symlink("../baz", "foo/bar2"),
		fstest.CreateFile("foo/bar1", nil, 0600),
		fstest.CreateFile("foo/bar3", nil, 0600),
		fstest.CreateFile("baz", nil, 0600),
	)
	assert.NilError(t, apply.Apply(tmpDir.Path()))

	out, err := FollowLinks(tmpDir.Path(), []string{"dir/l*"})
	assert.NilError(t, err)

	assert.Equal(t, out, []string{"baz", "dir/l*", "foo/bar1", "foo/bar2"})

	out, err = FollowLinks(tmpDir.Path(), []string{"dir"})
	assert.NilError(t, err)

	assert.Equal(t, out, []string{"dir"})

	out, err = FollowLinks(tmpDir.Path(), []string{"dir", "dir/*link"})
	assert.NilError(t, err)

	assert.Equal(t, out, []string{"dir", "foo/bar3"})
}
