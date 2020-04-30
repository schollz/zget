package links

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

var indexhtml = []byte(`<html><head></head><body><a href="https://schollz.com/blog">blog</a><a href="static/image.jpg">image</a></body></html>`)

func TestFindLinks(t *testing.T) {
	ioutil.WriteFile("index.html", indexhtml, 0644)
	links, err := FromFile("index.html", "https://schollz.com/blog/worker-pool/", true)
	assert.Nil(t, err)
	fmt.Println(links)
	b, _ := ioutil.ReadFile("index.html")
	assert.Equal(t, `<html><head></head><body><a href="/blog">blog</a><a href="/blog/worker-pool/static/image.jpg">image</a></body></html>`, string(b))
}
