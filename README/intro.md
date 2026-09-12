A function that calls `os.Create` has decided, permanently, that it writes to the
real filesystem. Testing it means a temporary directory, cleanup, and the quiet
possibility of one test seeing another's leftovers. A function that takes an
`afero.Fs` has decided nothing: production passes `afero.NewOsFs()`, a test passes
`afero.NewMemMapFs()`, and the same code runs against both.

That is the convention this module exists to make cheap. `afero` supplies the
filesystem abstraction and the operations the standard library has; `fsutil` adds
the ones it does not, in the same shape — filesystem first, paths after.

```go
// Reaches for the disk, whatever the caller wanted.
func writeReport(path string) error

// Writes wherever it is told to.
func writeReport(fs afero.Fs, path string) error
```
