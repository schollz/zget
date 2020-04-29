package torrent

import (
	"testing"

	log "github.com/schollz/logger"
	"github.com/stretchr/testify/assert"
)

func TestTorrent(t *testing.T) {
	log.SetLevel("trace")
	assert.Nil(t, downloadTorrent("magnet:?xt=urn:btih:d7ed8702f74b6db246abf75b78fe1cee3addd405&dn=enwiki-20200401-pages-articles-multistream.xml.bz2&ws=https%3a%2f%2fdumps.wikimedia.org%2fenwiki%2f20200401%2fenwiki-20200401-pages-articles-multistream.xml.bz2&ws=https%3a%2f%2fdumps.wikimedia.your.org%2fenwiki%2f20200401%2fenwiki-20200401-pages-articles-multistream.xml.bz2&ws=https%3a%2f%2fftp.acc.umu.se%2fmirror%2fwikimedia.org%2fdumps%2fenwiki%2f20200401%2fenwiki-20200401-pages-articles-multistream.xml.bz2"))
}
