package osutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MarkRosemaker/fsutil/osutil"
)

// Copy works on the real filesystem: the copy lands on disk, directories and all.
func TestCopy(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")

	if err := os.WriteFile(src, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(dir, "build", "dst.txt")
	if err := osutil.Copy(src, dst); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != "hello" {
		t.Errorf("copied %q, want hello", got)
	}
}
