package domain

import (
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
)

// ScoreVisibility measures answer text and citation URLs independently. Citations win.
func ScoreVisibility(brand BrandSnapshot, answer string, sources []string) Outcome {
	for _, source := range sources {
		u, err := url.Parse(source)
		if err != nil || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") {
			continue
		}

		host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
		domain := strings.TrimPrefix(strings.ToLower(brand.Domain), "www.")
		if domain != "" && (host == domain || strings.HasSuffix(host, "."+domain)) {
			return OutcomeCited
		}
		if brand.GitHubRepo != "" && host == "github.com" {
			repo, err := url.Parse(brand.GitHubRepo)
			if err == nil {
				path := strings.ToLower(strings.TrimSuffix(repo.Path, "/"))
				citedPath := strings.ToLower(u.Path)
				if citedPath == path || strings.HasPrefix(citedPath, path+"/") {
					return OutcomeCited
				}
			}
		}
	}

	name := foldText(brand.Name)
	if name == "" {
		return OutcomeMiss
	}

	if containsWord(foldText(answer), name) {
		return OutcomeMentioned
	}

	return OutcomeMiss
}

// foldText normalizes whitespace and case before name matching.
func foldText(value string) string {
	return cases.Fold().String(normalizeWhitespace(value))
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_'
}

// containsWord finds a folded name in folded text. A match is rejected when a word-like
// name edge touches a word-like neighbor, so "koi" does not match inside "koikoi".
func containsWord(text, name string) bool {
	if name == "" {
		return false
	}

	first, _ := utf8.DecodeRuneInString(name)
	last, _ := utf8.DecodeLastRuneInString(name)
	checkBefore := isWordRune(first)
	checkAfter := isWordRune(last)

	for offset := 0; offset <= len(text)-len(name); {
		index := strings.Index(text[offset:], name)
		if index < 0 {
			return false
		}

		start := offset + index
		end := start + len(name)
		offset = start + 1

		if checkBefore && start > 0 {
			before, _ := utf8.DecodeLastRuneInString(text[:start])
			if isWordRune(before) {
				continue
			}
		}
		if checkAfter && end < len(text) {
			after, _ := utf8.DecodeRuneInString(text[end:])
			if isWordRune(after) {
				continue
			}
		}

		return true
	}

	return false
}
