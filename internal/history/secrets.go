package history

import (
	"math"
	"strings"
	"unicode"
)

// LooksSensitive performs a best-effort detection of credential-like text.
// It combines known prefixes, known structured formats and a conservative
// high-entropy heuristic; ordinary prose and URLs must never match.
func LooksSensitive(text string) bool {
	s := strings.TrimSpace(text)
	if s == "" || len(s) > 4096 {
		return false
	}
	if IsURL(s) {
		return false
	}
	if containsAny(s, "\n\r") {
		if strings.Contains(s, "-----BEGIN") && strings.Contains(s, "PRIVATE KEY-----") {
			return true
		}
		// Multi-line texts are judged line by line to keep prose safe.
		for _, line := range strings.FieldsFunc(s, func(r rune) bool { return r == '\n' || r == '\r' }) {
			if tokenLike(strings.TrimSpace(line)) {
				return true
			}
		}
		return false
	}
	return tokenLike(s)
}

func containsAny(s, chars string) bool {
	for _, r := range chars {
		if strings.ContainsRune(s, r) {
			return true
		}
	}
	return false
}

// tokenLike judges a single line: known credential shapes, JWT structure,
// PEM headers or a single long token-like word.
func tokenLike(s string) bool {
	if s == "" || len(s) > 4096 {
		return false
	}
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "bearer ") {
		rest := strings.TrimSpace(s[len("Bearer "):])
		if containsAny(rest, " \t") || len(rest) < 16 {
			return false
		}
		return len(rest) >= 16 && hasLetters(rest) && highEntropy(rest)
	}
	if containsAny(s, " \t") {
		return false
	}
	switch {
	case strings.HasPrefix(lower, "ghp_"), strings.HasPrefix(lower, "gho_"),
		strings.HasPrefix(lower, "ghu_"), strings.HasPrefix(lower, "ghs_"),
		strings.HasPrefix(lower, "github_pat_"), strings.HasPrefix(lower, "sk-"),
		strings.HasPrefix(lower, "sk-ant-"), strings.HasPrefix(lower, "xoxb-"),
		strings.HasPrefix(lower, "xoxp-"), strings.HasPrefix(lower, "xoxa-"):
		return suffixEntropyOK(s)
	case strings.HasPrefix(s, "AKIA") && len(s) >= 20:
		return upperAlnum(s[4:])
	}
	if looksLikeJWT(s) {
		return true
	}
	if strings.HasPrefix(s, "-----BEGIN") && strings.Contains(s, "KEY-----") {
		return true
	}
	return highEntropyToken(s)
}

func suffixEntropyOK(s string) bool {
	rest := s
	if i := strings.IndexByte(s, '_'); i >= 0 && strings.HasPrefix(strings.ToLower(s), "github_pat_") {
		rest = s[i+1:]
	} else if i := strings.IndexByte(s, '-'); i >= 0 {
		rest = s[i+1:]
	}
	return len(rest) >= 16 && hasLetters(rest) && highEntropy(rest)
}

func upperAlnum(s string) bool {
	for _, r := range s {
		if !(r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func hasLetters(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func looksLikeJWT(s string) bool {
	if !strings.HasPrefix(s, "eyJ") || strings.Count(s, ".") != 2 {
		return false
	}
	parts := strings.Split(s, ".")
	return len(parts[1]) >= 20 && len(parts[2]) >= 10 && tokenCharset(parts[0]) && tokenCharset(parts[1])
}

func tokenCharset(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' && r != '=' {
			return false
		}
	}
	return true
}

// highEntropyToken flags a single unpunctuated token of at least 32 chars
// with mixed letters/digits/case and high entropy. Prose (spaces, sentence
// punctuation) and URLs never reach this check.
func highEntropyToken(s string) bool {
	if len(s) < 32 || !tokenCharset(s) {
		return false
	}
	hasDigit, hasLower, hasUpper := false, false, false
	for _, r := range s {
		switch {
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		}
	}
	if !hasDigit || (!hasLower && !hasUpper) {
		return false
	}
	return entropy(s) >= 3.5
}

func entropy(s string) float64 {
	counts := map[rune]int{}
	for _, r := range s {
		counts[r]++
	}
	var result float64
	for _, c := range counts {
		p := float64(c) / float64(len(s))
		result -= p * math.Log2(p)
	}
	return result
}

func highEntropy(s string) bool {
	return entropy(s) >= 3.0
}
