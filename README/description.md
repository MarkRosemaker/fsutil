---
tagline: Take a filesystem, not a path.
---

<div align="center" id=badges>


</div>



`fsutil` provides file operations written against
[`afero.Fs`](https://github.com/spf13/afero) rather than against the real
filesystem, so the code that calls them can be tested without touching a disk.

> **Status: early.** Only copying is here so far. The rest arrives as each
> operation earns its place.
