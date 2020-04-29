# zget

<a href="https://github.com/schollz/zget/releases/latest"><img src="https://img.shields.io/badge/version-v1.0.0-brightgreen.svg?style=flat-square" alt="Version"></a>

zget is basically a mashup of `curl` and `wget`. There are no real new features, but follows some things that I personally like, such as:

- Fast on Windows. For some reason `wget` (downloaded form scoop) is not fast on Windows.
- Uses `curl`-like arguments so its compatiable  with `Copy as cURL`. I don't like `curl` but `wget` has different arguments.
- Incorporates [httpstat](https://github.com/davecheney/httpstat) with `-stat` flag.
- Uses a progressbar instead of showing tons of lines.
- Hackable.

Features that I've added:

- [x] Allows multiple workers with `-w` flag.
- [x] Allows using tor with `-tor` flag.
- [x] Write compressed version to disk with `--gzip` flag
- [ ] Strip out style tags and/or script tags with `--strip-scripts` flag
- [x] Support to download magnet links 

# License 

MIT