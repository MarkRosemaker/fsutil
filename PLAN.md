### Repository Structure

```bash
fsutil/
├── fsutil/          # Main package: everything that takes afero.Fs
│   ├── fsutil.go
│   ├── cross.go
│   ├── copydir.go
│   ├── atomic.go
│   ├── mkdir.go
│   └── doc.go
├── osutil/              # Convenience functions for real OS filesystem
│   └── os.go
├── fstest/          # Testing helpers
│   └── fs.go
├── go.mod
├── README.md
└── fsutil_test.go   # Example tests
```

---

### 1. `fsutil/fsutil.go`

```go
// Package fsutil provides utilities for working with afero.Fs (and real OS filesystem via the os subpackage).
package fsutil

import (
	"github.com/spf13/afero"
	"io"
	"path/filepath"
)

// Copy copies a single file within the same filesystem.
func Copy(fs afero.Fs, src, dst string) error {
	return copyFile(fs, fs, src, dst)
}

// copyFile is the internal shared implementation.
func copyFile(srcFS, dstFS afero.Fs, src, dst string) error {
	if err := MkdirAll(dstFS, filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	srcFile, err := srcFS.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := dstFS.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Best-effort permission preservation
	if fi, err := srcFile.Stat(); err == nil {
		_ = dstFS.Chmod(dst, fi.Mode().Perm())
	}

	return dstFile.Sync()
}
```

---

### 2. `fsutil/cross.go`

```go
package fsutil

import "github.com/spf13/afero"

// CopyBetween copies a file from one filesystem to another (supports different FS types).
func CopyBetween(srcFS, dstFS afero.Fs, src, dst string) error {
	return copyFile(srcFS, dstFS, src, dst)
}

// CopyToOs copies a file from any FS into the real operating system.
func CopyToOs(srcFS afero.Fs, src, dst string) error {
	return CopyBetween(srcFS, afero.NewOsFs(), src, dst)
}

// CopyFromOs copies a file from the real OS into any FS.
func CopyFromOs(dstFS afero.Fs, src, dst string) error {
	return CopyBetween(afero.NewOsFs(), dstFS, src, dst)
}
```

---

### 3. `fsutil/copydir.go`

```go
package fsutil

import (
	"github.com/spf13/afero"
	"os"
	"path/filepath"
)

// CopyDir recursively copies a directory (and all its contents) from srcFS to dstFS.
func CopyDir(srcFS, dstFS afero.Fs, src, dst string) error {
	return afero.Walk(srcFS, src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := afero.Rel(srcFS, src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return MkdirAll(dstFS, target, info.Mode().Perm())
		}
		return copyFile(srcFS, dstFS, path, target)
	})
}
```

---

### 4. `fsutil/mkdir.go`

```go
package fsutil

import "github.com/spf13/afero"

// MkdirAll creates a directory and all parent directories.
func MkdirAll(fs afero.Fs, path string, perm os.FileMode) error {
	return fs.MkdirAll(path, perm)
}
```

---

### 5. `fsutil/atomic.go` (bonus)

```go
package fsutil

import (
	"github.com/spf13/afero"
	"os"
)

// WriteFileAtomic writes data to a file using a temporary file + rename (atomic on same FS).
func WriteFileAtomic(fs afero.Fs, name string, data []byte, perm os.FileMode) error {
	f, err := afero.TempFile(fs, filepath.Dir(name), ".tmp-"+filepath.Base(name))
	if err != nil {
		return err
	}
	tmpName := f.Name()
	defer fs.Remove(tmpName) // cleanup on failure

	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	if err := fs.Chmod(tmpName, perm); err != nil {
		return err
	}

	return fs.Rename(tmpName, name)
}
```

---

### 6. `fsutil/fstest/fs.go`

```go
// Package fstest provides testing helpers for filesystem assertions.
package fstest

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/afero"
)

var errMismatch = os.ErrInvalid // used only internally

// Equal asserts that two filesystems have identical structure and content.
func Equal(t *testing.T, fs1, fs2 afero.Fs, checkPermissions bool, msgAndArgs ...interface{}) {
	t.Helper()

	equal, err := equalRecursive(fs1, fs2, ".", checkPermissions)
	if err != nil {
		t.Fatalf("failed to compare filesystems: %v", err)
	}
	if !equal {
		t.Error("filesystems are not equal")
		if len(msgAndArgs) > 0 {
			t.Errorf(msgAndArgs[0].(string), msgAndArgs[1:]...)
		}
	}
}

func equalRecursive(fs1, fs2 afero.Fs, root string, checkPerms bool) (bool, error) {
	err := afero.Walk(fs1, root, func(path string, info1 os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		info2, err := fs2.Stat(path)
		if err != nil {
			return errMismatch // missing file
		}

		if info1.IsDir() != info2.IsDir() {
			return errMismatch
		}

		if info1.IsDir() {
			return nil
		}

		// Compare content
		b1, err := afero.ReadFile(fs1, path)
		if err != nil {
			return err
		}
		b2, err := afero.ReadFile(fs2, path)
		if err != nil {
			return err
		}
		if !bytes.Equal(b1, b2) {
			return errMismatch
		}

		if checkPerms && info1.Mode().Perm() != info2.Mode().Perm() {
			return errMismatch
		}

		return nil
	})

	if err == errMismatch {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// TODO: Optional second walk to ensure fs2 has no extra files
	return true, nil
}
```

---

### 7. `fsutil/os/os.go`

```go
// Package os provides convenient functions that operate directly on the real OS filesystem.
package os

import (
	"github.com/spf13/afero"
	"github.com/yourname/fsutil" // ← change to your actual module path
)

func Copy(src, dst string) error {
	return fsutil.Copy(afero.NewOsFs(), src, dst)
}

func CopyDir(src, dst string) error {
	return fsutil.CopyDir(afero.NewOsFs(), afero.NewOsFs(), src, dst)
}

func CopyToOs(srcFS afero.Fs, src, dst string) error {
	return fsutil.CopyToOs(srcFS, src, dst)
}

func CopyFromOs(dstFS afero.Fs, src, dst string) error {
	return fsutil.CopyFromOs(dstFS, src, dst)
}

// Add more helpers as needed (MkdirAll, WriteFileAtomic, etc.)
```
