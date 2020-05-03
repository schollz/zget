package links

import (
	"bytes"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	log "github.com/schollz/logger"
	"github.com/schollz/zget/src/utils"
)

func relativeURL(u1 *url.URL, u2 *url.URL) (relativeU string) {
	if u1.Path == "/" {
		relativeU = "." + u2.Path
		return
	}

	u1len := len(strings.Split(strings.TrimSuffix(strings.TrimPrefix(u1.Path, "/"), "/"), "/"))
	relativeU = strings.Repeat("../", u1len) + strings.TrimPrefix(u2.Path, "/")
	return
}

// FromFile retrieves, parses, and validates all links for given host
func FromFile(fname string, host string, rewrite bool) (links []string, err error) {
	b, err := ioutil.ReadFile(fname)
	if err != nil {
		return
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(b))
	if err != nil {
		return
	}

	uhost, err := utils.ParseURL(host)
	if err != nil {
		return
	}

	doc.Find("link").Each(func(i int, s *goquery.Selection) {
		link := s.AttrOr("href", "")
		if link == "" {
			return
		}
		u, errL := utils.FixURL(link, host)
		if errL != nil {
			log.Debug(errL)
			return
		}
		if u.Host == uhost.Host {
			links = append(links, relativeURL(uhost, u))
			s.SetAttr("href", strings.TrimPrefix(relativeURL(uhost, u), uhost.Scheme+"://"+uhost.Host))
		}
	})

	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		link := s.AttrOr("src", "")
		if link == "" {
			return
		}
		u, errL := utils.FixURL(link, host)
		if errL != nil {
			log.Debug(errL)
			return
		}
		if u.Host == uhost.Host {
			links = append(links, relativeURL(uhost, u))
			s.SetAttr("src", strings.TrimPrefix(relativeURL(uhost, u), uhost.Scheme+"://"+uhost.Host))

		}
	})

	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		link := s.AttrOr("src", "")
		if link == "" {
			return
		}
		u, errL := utils.FixURL(link, host)
		if errL != nil {
			log.Debug(errL)
			return
		}
		if u.Host == uhost.Host {
			links = append(links, relativeURL(uhost, u))
			s.SetAttr("src", strings.TrimPrefix(relativeURL(uhost, u), uhost.Scheme+"://"+uhost.Host))
		}
	})

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		link := s.AttrOr("href", "")
		if link == "" {
			return
		}
		u, errL := utils.FixURL(link, host)
		if errL != nil {
			log.Debug(errL)
			return
		}
		if u.Host == uhost.Host {
			links = append(links, relativeURL(uhost, u))
			s.SetAttr("href", strings.TrimPrefix(relativeURL(uhost, u), uhost.Scheme+"://"+uhost.Host))
		}
	})

	if rewrite {
		var html string
		html, err = doc.Html()
		if err != nil {
			return
		}
		err = ioutil.WriteFile(fname, []byte(html), 0644)
	}
	return
}
