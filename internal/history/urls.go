package history

import (
	"net/url"
	"strings"
)

// Classify detects the content type of a text clipboard item. Only a whole
// trimmed string that is a valid http(s) URL counts as a URL; texts that
// merely contain a link stay text.
func Classify(text string) ContentType {
	if IsURL(text) {
		return ContentURL
	}
	return ContentText
}

func IsURL(text string) bool {
	s := strings.TrimSpace(text)
	if s == "" || len(s) > 2048 || strings.ContainsAny(s, " \t\r\n\x00<>\"{}\\^`") {
		return false
	}
	parsed, e := url.Parse(s)
	if e != nil || parsed.Host == "" {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	// Reject credentials embedded in URLs and port-only hosts.
	if parsed.User != nil || strings.HasPrefix(parsed.Host, ":") {
		return false
	}
	return parsed.RequestURI() != ""
}
