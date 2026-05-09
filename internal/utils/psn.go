package utils

import (
	"net/url"
	"regexp"
	"strings"
)

var psnIDPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{2,15}$`)

func NormalizeOnlineID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func ValidOnlineID(value string) bool {
	return psnIDPattern.MatchString(strings.TrimSpace(value))
}

func ProfileURL(onlineID string) string {
	return "https://profile.playstation.com/" + url.PathEscape(strings.TrimSpace(onlineID))
}
