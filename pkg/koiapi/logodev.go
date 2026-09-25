package koiapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type CompetitorSearchResult struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

type CompetitorSearcher interface {
	Search(context.Context, string) ([]CompetitorSearchResult, error)
}

type LogoDevSearcher struct {
	apiKey string
	client *http.Client
}

func NewLogoDevSearcher(apiKey string) *LogoDevSearcher {
	return &LogoDevSearcher{apiKey: strings.TrimSpace(apiKey), client: &http.Client{Timeout: 5 * time.Second}}
}

func (s *LogoDevSearcher) Search(ctx context.Context, query string) ([]CompetitorSearchResult, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("LOGODEV_SECRET_KEY is not configured")
	}
	endpoint := "https://api.logo.dev/search?q=" + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("logo.dev search returned %s", resp.Status)
	}
	var results []CompetitorSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("decode logo.dev search: %w", err)
	}
	usable := make([]CompetitorSearchResult, 0, min(len(results), 10))
	for _, result := range results {
		if result.Name == "" || result.Domain == "" || strings.ContainsAny(result.Domain, "/?#") {
			continue
		}
		usable = append(usable, result)
		if len(usable) == 10 {
			break
		}
	}
	return usable, nil
}
