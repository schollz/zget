package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestU(t *testing.T) {
	u, err := FixURL("something.jpg", "schollz.com/a")
	assert.Nil(t, err)
	assert.Equal(t, "https://schollz.com/a/something.jpg", u.String())

	u, err = FixURL("/something.jpg", "schollz.com/a")
	assert.Nil(t, err)
	assert.Equal(t, "https://schollz.com/something.jpg", u.String())

	u, err = FixURL("./something.jpg", "schollz.com/a")
	assert.Nil(t, err)
	assert.Equal(t, "https://schollz.com/a/something.jpg", u.String())

	u, err = FixURL("../blog/", "schollz.com/a")
	assert.Nil(t, err)
	assert.Equal(t, "https://schollz.com/blog/", u.String())

}
