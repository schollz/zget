package splicer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
)

func TestSplicer(t *testing.T) {
	assert.Nil(t, try())

}

func try() (err error) {
	m := minify.New()
	m.Add("text/html", &html.Minifier{
		KeepDefaultAttrVals: true,
		KeepDocumentTags:    true,
	})
	var data = `
<html>
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
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(data))
	if err != nil {
		return
	}

	doc.Find("script").ReplaceWithHtml("")
	html, err := doc.Html()
	if err != nil {
		return
	}
	fmt.Println(html)
	s, err := m.String("text/html", html)
	if err != nil {
		return
	}
	fmt.Println(s)

	return
}
