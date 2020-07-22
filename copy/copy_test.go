package fs

import (
	"context"
	_ "crypto/sha256"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/containerd/continuity/fs/fstest"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
)

// TODO: Create copy directory which requires privilege
//  chown
//  mknod
//  setxattr fstest.SetXAttr("/home", "trusted.overlay.opaque", "y"),

func TestCopyDirectory(t *testing.T) {
	apply := fstest.Apply(
		fstest.CreateDir("/etc/", 0755),
		fstest.CreateFile("/etc/hosts", []byte("localhost 127.0.0.1"), 0644),
		fstest.Link("/etc/hosts", "/etc/hosts.allow"),
		fstest.CreateDir("/usr/local/lib", 0755),
		fstest.CreateFile("/usr/local/lib/libnothing.so", []byte{0x00, 0x00}, 0755),
		fstest.Symlink("libnothing.so", "/usr/local/lib/libnothing.so.2"),
		fstest.CreateDir("/home", 0755),
	)

	assert.NilError(t, testCopy(apply))
}

// This test used to fail because link-no-nothing.txt would be copied first,
// then file operations in dst during the CopyDir would follow the symlink and
// fail.
func TestCopyDirectoryWithLocalSymlink(t *testing.T) {
	apply := fstest.Apply(
		fstest.CreateFile("nothing.txt", []byte{0x00, 0x00}, 0755),
		fstest.Symlink("nothing.txt", "link-no-nothing.txt"),
	)

	assert.NilError(t, testCopy(apply))
}

func TestCopyToWorkDir(t *testing.T) {
	t1, err := ioutil.TempDir("", "test")
	assert.NilError(t, err)
	defer os.RemoveAll(t1)

	apply := fstest.Apply(
		fstest.CreateFile("foo.txt", []byte("contents"), 0755),
	)

	assert.NilError(t, apply.Apply(t1))

	t2, err := ioutil.TempDir("", "test")
	assert.NilError(t, err)
	defer os.RemoveAll(t2)

	err = Copy(context.TODO(), t1, "foo.txt", t2, "foo.txt")
	assert.NilError(t, err)

	err = fstest.CheckDirectoryEqual(t1, t2)
	assert.NilError(t, err)
}

func TestCopySingleFile(t *testing.T) {
	t1, err := ioutil.TempDir("", "test")
	assert.NilError(t, err)
	defer os.RemoveAll(t1)

	apply := fstest.Apply(
		fstest.CreateFile("foo.txt", []byte("contents"), 0755),
	)

	assert.NilError(t, apply.Apply(t1))

	t2, err := ioutil.TempDir("", "test")
	assert.NilError(t, err)
	defer os.RemoveAll(t2)

	err = Copy(context.TODO(), t1, "foo.txt", t2, "/")
	assert.NilError(t, err)

	err = fstest.CheckDirectoryEqual(t1, t2)
	assert.NilError(t, err)

	t3, err := ioutil.TempDir("", "test")
	assert.NilError(t, err)
	defer os.RemoveAll(t2)

	err = Copy(context.TODO(), t1, "foo.txt", t3, "foo.txt")
	assert.NilError(t, err)

	err = fstest.CheckDirectoryEqual(t1, t2)
	assert.NilError(t, err)

	t4, err := ioutil.TempDir("", "test")
	assert.NilError(t, err)
	defer os.RemoveAll(t2)

	err = Copy(context.TODO(), t1, "foo.txt", t4, "foo2.txt")
	assert.NilError(t, err)

	_, err = os.Stat(filepath.Join(t4, "foo2.txt"))
	assert.NilError(t, err)
}

func TestCopyOverrideFile(t *testing.T) {
	t1, err := ioutil.TempDir("", "test")
	assert.NilError(t, err)
	defer os.RemoveAll(t1)

	apply := fstest.Apply(
		fstest.CreateFile("foo.txt", []byte("contents"), 0755),
	)

	assert.NilError(t, apply.Apply(t1))

	t2, err := ioutil.TempDir("", "test")
	assert.NilError(t, err)
	defer os.RemoveAll(t2)

	err = Copy(context.TODO(), t1, "foo.txt", t2, "foo.txt")
	assert.NilError(t, err)

	err = fstest.CheckDirectoryEqual(t1, t2)
	assert.NilError(t, err)

	err = Copy(context.TODO(), t1, "foo.txt", t2, "foo.txt")
	assert.NilError(t, err)

	err = fstest.CheckDirectoryEqual(t1, t2)
	assert.NilError(t, err)

	err = Copy(context.TODO(), t1, "/.", t2, "/")
	assert.NilError(t, err)

	err = fstest.CheckDirectoryEqual(t1, t2)
	assert.NilError(t, err)
}

func TestCopyDirectoryBasename(t *testing.T) {
	t1, err := ioutil.TempDir("", "test")
	assert.NilError(t, err)
	defer os.RemoveAll(t1)

	apply := fstest.Apply(
		fstest.CreateDir("foo", 0755),
		fstest.CreateDir("foo/bar", 0755),
		fstest.CreateFile("foo/bar/baz.txt", []byte("contents"), 0755),
	)
	assert.NilError(t, apply.Apply(t1))

	t2, err := ioutil.TempDir("", "test")
	assert.NilError(t, err)
	defer os.RemoveAll(t2)

	err = Copy(context.TODO(), t1, "foo", t2, "foo")
	assert.NilError(t, err)

	err = fstest.CheckDirectoryEqual(t1, t2)
	assert.NilError(t, err)

	err = Copy(context.TODO(), t1, "foo", t2, "foo", WithCopyInfo(CopyInfo{
		CopyDirContents: true,
	}))
	assert.NilError(t, err)

	err = fstest.CheckDirectoryEqual(t1, t2)
	assert.NilError(t, err)
}

func TestCopyWildcards(t *testing.T) {
	t1, err := ioutil.TempDir("", "test")
	assert.NilError(t, err)
	defer os.RemoveAll(t1)

	apply := fstest.Apply(
		fstest.CreateFile("foo.txt", []byte("foo-contents"), 0755),
		fstest.CreateFile("foo.go", []byte("go-contents"), 0755),
		fstest.CreateFile("bar.txt", []byte("bar-contents"), 0755),
	)

	assert.NilError(t, apply.Apply(t1))

	t2, err := ioutil.TempDir("", "test")
	assert.NilError(t, err)
	defer os.RemoveAll(t2)

	err = Copy(context.TODO(), t1, "foo*", t2, "/")
	assert.Assert(t, err != nil)

	err = Copy(context.TODO(), t1, "foo*", t2, "/", AllowWildcards)
	assert.NilError(t, err)

	_, err = os.Stat(filepath.Join(t2, "foo.txt"))
	assert.NilError(t, err)
	_, err = os.Stat(filepath.Join(t2, "foo.go"))
	assert.NilError(t, err)
	_, err = os.Stat(filepath.Join(t2, "bar.txt"))
	assert.Assert(t, os.IsNotExist(err))

	t2, err = ioutil.TempDir("", "test")
	assert.NilError(t, err)
	defer os.RemoveAll(t2)

	err = Copy(context.TODO(), t1, "bar*", t2, "foo.txt", AllowWildcards)
	assert.NilError(t, err)
	dt, err := ioutil.ReadFile(filepath.Join(t2, "foo.txt"))
	assert.NilError(t, err)
	assert.Equal(t, "bar-contents", string(dt))
}

func TestCopyExistingDirDest(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip()
	}

	t1, err := ioutil.TempDir("", "test")
	assert.NilError(t, err)
	defer os.RemoveAll(t1)

	apply := fstest.Apply(
		fstest.CreateDir("dir", 0755),
		fstest.CreateFile("dir/foo.txt", []byte("foo-contents"), 0644),
		fstest.CreateFile("dir/bar.txt", []byte("bar-contents"), 0644),
	)
	assert.NilError(t, apply.Apply(t1))

	t2, err := ioutil.TempDir("", "test")
	assert.NilError(t, err)
	defer os.RemoveAll(t2)

	apply = fstest.Apply(
		// notice how perms for destination and source are different
		fstest.CreateDir("dir", 0700),
		// dir/foo.txt does not exist, but dir/bar.txt does
		// notice how both perms and contents for destination and source are different
		fstest.CreateFile("dir/bar.txt", []byte("old-bar-contents"), 0600),
	)
	assert.NilError(t, apply.Apply(t2))

	for _, x := range []string{"dir", "dir/bar.txt"} {
		err = os.Chown(filepath.Join(t2, x), 1, 1)
		assert.NilErrorf(t, err, "x=%s", x)
	}

	err = Copy(context.TODO(), t1, "dir", t2, "dir", WithCopyInfo(CopyInfo{
		CopyDirContents: true,
	}))
	assert.NilError(t, err)

	// verify that existing destination dir's metadata was not overwritten
	st, err := os.Lstat(filepath.Join(t2, "dir"))
	assert.NilError(t, err)
	assert.Equal(t, st.Mode()&os.ModePerm, os.FileMode(0700))
	uid, gid := getUidGid(st)
	assert.Equal(t, 1, uid)
	assert.Equal(t, 1, gid)

	// verify that non-existing file was created
	_, err = os.Lstat(filepath.Join(t2, "dir/foo.txt"))
	assert.NilError(t, err)

	// verify that existing file's content and metadata was overwritten
	st, err = os.Lstat(filepath.Join(t2, "dir/bar.txt"))
	assert.NilError(t, err)
	assert.Equal(t, os.FileMode(0644), st.Mode()&os.ModePerm)
	uid, gid = getUidGid(st)
	assert.Equal(t, 0, uid)
	assert.Equal(t, 0, gid)
	dt, err := ioutil.ReadFile(filepath.Join(t2, "dir/bar.txt"))
	assert.NilError(t, err)
	assert.Equal(t, "bar-contents", string(dt))
}

func TestCopySymlinks(t *testing.T) {
	t1, err := ioutil.TempDir("", "test")
	assert.NilError(t, err)
	defer os.RemoveAll(t1)

	apply := fstest.Apply(
		fstest.CreateDir("testdir", 0755),
		fstest.CreateFile("testdir/foo.txt", []byte("foo-contents"), 0644),
		fstest.Symlink("foo.txt", "testdir/link2"),
		fstest.Symlink("/testdir", "link"),
	)
	assert.NilError(t, apply.Apply(t1))

	t2, err := ioutil.TempDir("", "test")
	assert.NilError(t, err)
	defer os.RemoveAll(t2)

	err = Copy(context.TODO(), t1, "link/link2", t2, "foo", WithCopyInfo(CopyInfo{
		FollowLinks: true,
	}))
	assert.NilError(t, err)

	// verify that existing destination dir's metadata was not overwritten
	st, err := os.Lstat(filepath.Join(t2, "foo"))
	assert.NilError(t, err)
	assert.Equal(t, os.FileMode(0644), st.Mode()&os.ModePerm)
	assert.Equal(t, 0, int(st.Mode()&os.ModeSymlink))
	dt, err := ioutil.ReadFile(filepath.Join(t2, "foo"))
	assert.Equal(t, "foo-contents", string(dt))

	t3, err := ioutil.TempDir("", "test")
	assert.NilError(t, err)
	defer os.RemoveAll(t2)

	err = Copy(context.TODO(), t1, "link/link2", t3, "foo", WithCopyInfo(CopyInfo{}))
	assert.NilError(t, err)

	// verify that existing destination dir's metadata was not overwritten
	st, err = os.Lstat(filepath.Join(t3, "foo"))
	assert.NilError(t, err)
	assert.Equal(t, os.ModeSymlink, st.Mode()&os.ModeSymlink)
	link, err := os.Readlink(filepath.Join(t3, "foo"))
	assert.NilError(t, err)
	assert.Equal(t, "foo.txt", link)
}

func testCopy(apply fstest.Applier) error {
	t1, err := ioutil.TempDir("", "test-copy-src-")
	if err != nil {
		return errors.Wrap(err, "failed to create temporary directory")
	}
	defer os.RemoveAll(t1)

	t2, err := ioutil.TempDir("", "test-copy-dst-")
	if err != nil {
		return errors.Wrap(err, "failed to create temporary directory")
	}
	defer os.RemoveAll(t2)

	if err := apply.Apply(t1); err != nil {
		return errors.Wrap(err, "failed to apply changes")
	}

	if err := Copy(context.TODO(), t1, "/.", t2, "/"); err != nil {
		return errors.Wrap(err, "failed to copy")
	}

	return fstest.CheckDirectoryEqual(t1, t2)
}
