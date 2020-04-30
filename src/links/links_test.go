package links

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindLinks(t *testing.T) {
	links, err := FromFile("index.html", "https://schollz.com/blog/worker-pool/")
	assert.Nil(t, err)
	fmt.Println(links)
}
