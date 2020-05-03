package links

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

var indexhtml = []byte(`<html><head><script src="https://jquery.com/script/jquery.js"></script></head><body><img src="/img/pottery.jpg"><a href="https://schollz.com/blog/">blog</a><a href="static/image.jpg">image</a></body></html>`)

func TestFindLinks(t *testing.T) {
	ioutil.WriteFile("index.html", indexhtml, 0644)
	links, err := FromFile("index.html", "https://schollz.com/blog/worker-pool/", true)
	assert.Nil(t, err)
	fmt.Println(links)
	b, _ := ioutil.ReadFile("index.html")
	assert.Equal(t, `<html><head><script src="../../../jquery.com/script/jquery.js"></script></head><body><img src="/img/pottery.jpg"><a href="../../../schollz.com/blog/">blog</a><a href="../../../schollz.com/blog/worker-pool/static/image.jpg">image</a></body></html>`, string(b))
}
