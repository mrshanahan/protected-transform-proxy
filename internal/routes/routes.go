package routes

import (
	"regexp"
)

type RouteTransform func([]byte) []byte

type RouteConfig struct {
	Transform RouteTransform
}

var (
	Transforms map[string]*RouteConfig = map[string]*RouteConfig{
		"rwn-feed": {
			Transform: ReplaceRwnUrls,
		},
	}
	ExiledOnlineHttpPatt *regexp.Regexp = regexp.MustCompile(`http://exiledonline`)
)

func ReplaceRwnUrls(doc []byte) []byte {
	return ExiledOnlineHttpPatt.ReplaceAll(doc, []byte("https://exiledonline"))
}
