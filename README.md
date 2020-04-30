# zget

<a href="https://github.com/schollz/zget/releases/latest"><img src="https://img.shields.io/badge/version-v1.1.1-brightgreen.svg?style=flat-square" alt="Version"></a>

<p align="center"><code>curl https://getzget.schollz.com | bash</code></p>

zget is a mashup of `curl` and `wget`. I use `wget` on my Windows machine. But [`wget` is slow in Powershell](https://stackoverflow.com/questions/28682642/powershell-why-is-using-invoke-webrequest-much-slower-than-a-browser-download). Also, though I like most `wget` options, I'd like to use the `Copy as cURL` option which requires renaming flags from `wget` to `curl` (e.g. `-H` and `--compressed`). Sometimes I may find something else I want so I also want it to be hackable. The result is `zget`.

## Features

I've been adding features that aren't part of `curl` or `wget`, here are some of them.

- [x] Uses a progressbar instead of showing tons of lines.
- [x] Allows multiple workers with `-w` flag.
- [x] Allows using tor with `-tor` flag.
- [x] Write compressed version to disk with `--gzip` flag
- [x] Strip out style tags and/or script tags with `--rm-script/--rm-style` flag
- [x] Support to download magnet links 
- [x] Incorporates [httpstat](https://github.com/davecheney/httpstat) with `-stat` flag.

# License 

MIT
