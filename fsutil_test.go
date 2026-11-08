package fsutil_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/MarkRosemaker/fsutil"
	"github.com/spf13/afero"
)

func write(t *testing.T, fsys afero.Fs, name, content string, perm os.FileMode) {
	t.Helper()

	if err := afero.WriteFile(fsys, name, []byte(content), perm); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, fsys afero.Fs, name string) string {
	t.Helper()

	b, err := afero.ReadFile(fsys, name)
	if err != nil {
		t.Fatal(err)
	}

	return string(b)
}

// Copy creates the destination's missing directories and gives it the source's content and permissions, on the
// filesystem it is handed, whichever that is.
func TestCopy(t *testing.T) {
	t.Parallel()

	for name, fsys := range map[string]afero.Fs{
		"memory": afero.NewMemMapFs(),
		"disk":   afero.NewBasePathFs(afero.NewOsFs(), t.TempDir()),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			write(t, fsys, "src.txt", "hello", 0o640)

			if err := fsutil.Copy(fsys, "src.txt", "nested/deeper/dst.txt"); err != nil {
				t.Fatal(err)
			}

			if got := read(t, fsys, "nested/deeper/dst.txt"); got != "hello" {
				t.Errorf("copied %q, want hello", got)
			}

			fi, err := fsys.Stat("nested/deeper/dst.txt")
			if err != nil {
				t.Fatal(err)
			}

			if perm := fi.Mode().Perm(); perm != 0o640 {
				t.Errorf("copy has mode %v, want the source's -rw-r-----", perm)
			}

			if got := read(t, fsys, "src.txt"); got != "hello" {
				t.Errorf("the source now holds %q", got)
			}
		})
	}
}

// A destination that exists is replaced whole, not written over from the start.
func TestCopy_ReplacesDestination(t *testing.T) {
	t.Parallel()

	fsys := afero.NewMemMapFs()
	write(t, fsys, "src.txt", "new", 0o644)
	write(t, fsys, "dst.txt", "older and longer", 0o644)

	if err := fsutil.Copy(fsys, "src.txt", "dst.txt"); err != nil {
		t.Fatal(err)
	}

	if got := read(t, fsys, "dst.txt"); got != "new" {
		t.Errorf("destination holds %q, want new", got)
	}
}

func TestCopy_MissingSource(t *testing.T) {
	t.Parallel()

	fsys := afero.NewMemMapFs()

	err := fsutil.Copy(fsys, "missing.txt", "dst.txt")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("got %v, want fs.ErrNotExist", err)
	}

	if _, err := fsys.Stat("dst.txt"); err == nil {
		t.Error("created a destination for a source that does not exist")
	}
}

// A parent of the destination that is a file cannot become a directory.
func TestCopy_ParentIsAFile(t *testing.T) {
	t.Parallel()

	fsys := afero.NewBasePathFs(afero.NewOsFs(), t.TempDir())
	write(t, fsys, "src.txt", "hello", 0o644)
	write(t, fsys, "file", "", 0o644)

	if err := fsutil.Copy(fsys, "src.txt", "file/dst.txt"); err == nil {
		t.Error("copying under a file: want an error")
	}
}

// Copying a file onto itself would empty it, so it is refused and the file left alone, however it is named.
func TestCopy_SameFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	osFs := afero.NewOsFs()
	src := filepath.Join(dir, "a.txt")
	write(t, osFs, src, "hello", 0o644)

	link := filepath.Join(dir, "link.txt")
	if err := os.Link(src, link); err != nil {
		t.Fatal(err)
	}

	for _, dst := range []string{src, filepath.Join(dir, ".", "a.txt"), link} {
		if err := fsutil.Copy(osFs, src, dst); !errors.Is(err, fsutil.ErrSameFile) {
			t.Errorf("copying %s to %s: got %v, want ErrSameFile", src, dst, err)
		}

		if got := read(t, osFs, src); got != "hello" {
			t.Fatalf("copying %s to %s left %q", src, dst, got)
		}
	}
}

// On a filesystem that cannot tell whether two names are one file, the same name spelled two ways still is.
func TestCopy_SameFileInMemory(t *testing.T) {
	t.Parallel()

	fsys := afero.NewMemMapFs()
	write(t, fsys, "dir/a.txt", "hello", 0o644)

	if err := fsutil.Copy(fsys, "dir/a.txt", "dir/../dir/a.txt"); !errors.Is(err, fsutil.ErrSameFile) {
		t.Errorf("got %v, want ErrSameFile", err)
	}
}

func TestMkdirAll(t *testing.T) {
	t.Parallel()

	fsys := afero.NewMemMapFs()

	if err := fsutil.MkdirAll(fsys, "a/b/c", 0o750); err != nil {
		t.Fatal(err)
	}

	if ok, err := afero.DirExists(fsys, "a/b/c"); err != nil || !ok {
		t.Errorf("a/b/c is not a directory: %v", err)
	}
}
