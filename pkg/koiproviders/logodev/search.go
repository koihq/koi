package logodev

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/koihq/koi/pkg/koidomain"
)

const searchURL = "https://api.logo.dev/search"

type Searcher struct {
	key    string
	client *resty.Client
	url    string
}

func NewSearcher(key string) *Searcher {
	return &Searcher{key: strings.TrimSpace(key), client: resty.New().SetTimeout(5 * time.Second), url: searchURL}
}

func (s *Searcher) Search(ctx context.Context, query string) ([]koidomain.CompanySearchResult, error) {
	var results []koidomain.CompanySearchResult
	resp, err := s.client.R().SetContext(ctx).SetAuthToken(s.key).SetQueryParam("q", query).
		SetResult(&results).Get(s.url)
	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("logo.dev search returned %s", resp.Status())
	}
	usable := make([]koidomain.CompanySearchResult, 0, min(len(results), 10))
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

var _ koidomain.CompanySearcher = (*Searcher)(nil)
