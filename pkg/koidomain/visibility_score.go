package koidomain

import (
	"net/url"
	"strings"
	"unicode"

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

	fold := cases.Fold()
	name := []rune(fold.String(strings.Join(strings.Fields(brand.Name), " ")))
	text := []rune(fold.String(strings.Join(strings.Fields(answer), " ")))
	if len(name) == 0 {
		return OutcomeMiss
	}
	word := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' }
	for i := 0; i+len(name) <= len(text); i++ {
		if string(text[i:i+len(name)]) != string(name) {
			continue
		}
		if i > 0 && word(name[0]) && word(text[i-1]) {
			continue
		}
		end := i + len(name)
		if end < len(text) && word(name[len(name)-1]) && word(text[end]) {
			continue
		}

		return OutcomeMentioned
	}

	return OutcomeMiss
}
