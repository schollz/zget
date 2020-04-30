package splicer

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var data = `
<html>
<head>
<style>
p {
	text-size: 2em;
}
</style>
  <body>
  <script>
    var zodiac_content = {
    "data" : [
        {
             "type" : "headline",
             "text" : "TODAY'S STAR RATINGS"
        }
    ]}
  </script>
<strong>Hi</strong>
  <script src="javascript">
    var zodiac_content = {
    "data" : [
        {
             "type" : "headline",
             "text" : "TODAY'S STAR RATINGS"
        }
    ]}
  </script>
  </body>
</html>
`

func TestSplicer(t *testing.T) {
	defer os.Remove("temp.html")
	ioutil.WriteFile("temp.html", []byte(data), 0644)
	assert.Nil(t, StripHTML("temp.html", true, true))
	b, _ := ioutil.ReadFile("temp.html")
	assert.Equal(t, "<html><head></head><body><strong>Hi</strong></body></html>", string(b))

	ioutil.WriteFile("temp.html", []byte(data), 0644)
	assert.Nil(t, StripHTML("temp.html", true, false))
	b, _ = ioutil.ReadFile("temp.html")
	assert.Equal(t, `<html><head><style>
p {
	text-size: 2em;
}
</style></head><body><strong>Hi</strong></body></html>`, string(b))

}
