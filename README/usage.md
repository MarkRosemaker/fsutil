```bash
go get github.com/MarkRosemaker/fsutil
```

```go
import (
    "github.com/MarkRosemaker/fsutil"
    "github.com/spf13/afero"
)

// Copies within one filesystem, creating the destination directory as needed.
if err := fsutil.Copy(afero.NewOsFs(), "api/openapi.json", "build/openapi.json"); err != nil {
    log.Fatal(err)
}
```

The copy creates any missing parent directories, preserves the source file's
permissions on a best-effort basis, and syncs before returning.

In a test, the same call runs entirely in memory:

```go
fs := afero.NewMemMapFs()
afero.WriteFile(fs, "src.txt", []byte("hello"), 0o644)

if err := fsutil.Copy(fs, "src.txt", "nested/dst.txt"); err != nil {
    t.Fatal(err)
}
```

### When the filesystem is not in question

Command-line tools and code generators operate on the real filesystem by
definition, and threading `afero.NewOsFs()` through them buys nothing. The
`osutil` subpackage is the same operations with that argument already applied:

```go
import "github.com/MarkRosemaker/fsutil/osutil"

if err := osutil.Copy("api/openapi.json", "build/openapi.json"); err != nil {
    log.Fatal(err)
}
```

| Package | Signature | Use when |
|---|---|---|
| `fsutil` | `Copy(fs afero.Fs, src, dst string) error` | The caller should be able to choose the filesystem — which is most of the time |
| `fsutil/osutil` | `Copy(src, dst string) error` | The real filesystem is the whole point, as in a CLI or a generator |
