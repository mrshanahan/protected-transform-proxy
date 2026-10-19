package routes

import (
	"regexp"

	"github.com/mrshanahan/go-utils/auth"
)

type RouteTransform func([]byte) []byte

type RouteConfig struct {
	KeyHash   []byte
	KeySalt   []byte
	Transform RouteTransform
}

var (
	ExiledOnlineHttpPatt *regexp.Regexp = regexp.MustCompile(`http://exiledonline`)
	RwnFeedConfig                       = &RouteConfig{
		KeyHash:   auth.MustDecodeHex(""),
		KeySalt:   auth.MustDecodeHex(""),
		Transform: ReplaceRwnUrls,
	}
)

func ReplaceRwnUrls(doc []byte) []byte {
	return ExiledOnlineHttpPatt.ReplaceAll(doc, []byte("https://exiledonline"))
}
