package history

import "testing"

func TestClassifyURL(t *testing.T) {
	urls := []string{
		"https://example.com",
		"https://example.com/very/long/path?q=1&r=2",
		"http://localhost:45873/api/v1/health",
		"https://github.com/SawGoD/clipare",
	}
	for _, text := range urls {
		if got := Classify(text); got != ContentURL {
			t.Fatalf("%q classified as %q", text, got)
		}
	}
	texts := []string{
		"see https://example.com and reply",
		"Ссылка: https://example.com",
		"example.com",
		"ftp://files.example.com/archive.zip",
		"https://",
		"file:///etc/passwd",
		"docker compose ps",
		"",
	}
	for _, text := range texts {
		if got := Classify(text); got != ContentText {
			t.Fatalf("%q classified as %q", text, got)
		}
	}
}
