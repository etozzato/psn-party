package utils

import (
	"net/url"
	"regexp"
	"strings"
)

var psnIDPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{2,15}$`)
var onlineIDReplacer = strings.NewReplacer(
	"‐", "-",
	"‑", "-",
	"‒", "-",
	"–", "-",
	"—", "-",
	"―", "-",
	"−", "-",
	"﹣", "-",
	"－", "-",
	"＿", "_",
)

func CleanOnlineID(value string) string {
	return onlineIDReplacer.Replace(strings.TrimSpace(value))
}

func NormalizeOnlineID(value string) string {
	return strings.ToLower(CleanOnlineID(value))
}

func ValidOnlineID(value string) bool {
	return psnIDPattern.MatchString(CleanOnlineID(value))
}

func ProfileURL(onlineID string) string {
	return "https://profile.playstation.com/" + url.PathEscape(CleanOnlineID(onlineID))
}
