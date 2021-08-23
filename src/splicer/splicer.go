package splicer

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/PuerkitoBio/goquery"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/html"
)

var m *minify.M

func init() {
	fmt.Println("init")
	m = minify.New()
	m.Add("text/html", &html.Minifier{
		KeepDefaultAttrVals: true,
		KeepDocumentTags:    true,
	})
}

// StripHTML removes various components from HTML
func StripHTML(fname string, scriptTags bool, cssStyles bool) (err error) {
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		return
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		return
	}

	if scriptTags {
		doc.Find("script").ReplaceWithHtml("")
	}
	if cssStyles {
		doc.Find("style").ReplaceWithHtml("")
	}
	html, err := doc.Html()
	if err != nil {
		return
	}

	// minify
	s, err := m.String("text/html", html)
	if err != nil {
		return
	}

	// overwrite file
	err = ioutil.WriteFile(fname, []byte(s), 0644)
	return
}
