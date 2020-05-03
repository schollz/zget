package utils

import (
	"fmt"
	"math"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/purell"
)

func HumanizeBytes(s float64) string {
	sizes := []string{" B", " kB", " MB", " GB", " TB", " PB", " EB"}
	base := 1000.0
	if s < 10 {
		return fmt.Sprintf("%2.0f B", s)
	}
	e := math.Floor(logn(float64(s), base))
	suffix := sizes[int(e)]
	val := math.Floor(float64(s)/math.Pow(base, e)*10+0.5) / 10
	f := "%.0f%s"
	if val < 10 {
		f = "%.1f%s"
	}

	return fmt.Sprintf(f, val, suffix)
}

func logn(n, b float64) float64 {
	return math.Log(n) / math.Log(b)
}

func FixURL(uri string, sourceuri string) (u *url.URL, err error) {
	if !strings.Contains(sourceuri, "http") {
		sourceuri = "https://" + sourceuri
	}
	if !strings.HasSuffix(sourceuri, "/") {
		sourceuri += "/"
	}
	sourceu, err := ParseURL(sourceuri)
	if err != nil {
		return
	}

	if strings.HasPrefix(uri, "http") {
		// don't do anything
	} else if strings.HasPrefix(uri, "/") {
		uri = sourceu.Scheme + "://" + sourceu.Host + "/" + uri
	} else {
		uri = strings.TrimSuffix(sourceu.String(), "/") + "/" + uri
	}

	u, err = ParseURL(purell.MustNormalizeURLString(uri, purell.FlagRemoveDotSegments|purell.FlagRemoveDuplicateSlashes))
	if err != nil {
		return
	}

	return
}

func ParseURL(uri string) (*url.URL, error) {
	if !strings.Contains(uri, "://") && !strings.HasPrefix(uri, "//") {
		uri = "//" + uri
	}

	url, err := url.Parse(uri)
	if err != nil {
		return url, err
	}

	if url.Scheme == "" {
		url.Scheme = "http"
		if !strings.HasSuffix(url.Host, ":80") {
			url.Scheme += "s"
		}
	}
	return url, err
}
