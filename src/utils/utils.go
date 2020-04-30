package utils

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/purell"
)

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

	if strings.HasPrefix(uri, "/") {
		uri = sourceu.Scheme + "://" + sourceu.Host + "/" + uri
	} else if strings.HasPrefix(uri, "./") {
		uri = sourceu.String() + "/" + uri
	} else if !strings.HasPrefix(uri, "http") {
		uri = sourceu.String() + "/" + uri
	}

	u, err = ParseURL(purell.MustNormalizeURLString(uri, purell.FlagsUsuallySafeGreedy|purell.FlagRemoveDuplicateSlashes))
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
