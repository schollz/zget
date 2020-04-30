package links

import (
	"os"

	"github.com/PuerkitoBio/goquery"
	log "github.com/schollz/logger"
	"github.com/schollz/zget/src/utils"
)

// FromFile retrieves, parses, and validates all links for given host
func FromFile(fname string, host string) (links []string, err error) {
	f, err := os.Open(fname)
	if err != nil {
		return
	}
	defer f.Close()

	doc, err := goquery.NewDocumentFromReader(f)
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
			links = append(links, u.String())
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
			links = append(links, u.String())
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
			links = append(links, u.String())
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
			links = append(links, u.String())
		}
	})

	return
}
