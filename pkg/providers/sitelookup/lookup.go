// Package sitelookup reads a brand's public name and description from its website or GitHub repo.
package sitelookup

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/koihq/koi/pkg/domain"
)

const (
	maxPageBytes      = 256 << 10
	maxNameLength     = 40
	maxDescriptionLen = 300
)

var errBlockedAddress = errors.New("address is not public")

type Client struct {
	web       *http.Client
	api       *resty.Client
	webOrigin string
}

func New() *Client {
	dialer := &net.Dialer{Timeout: 3 * time.Second, Control: requirePublicAddress}

	return &Client{
		web: &http.Client{
			Timeout: 6 * time.Second,
			Transport: &http.Transport{
				// No proxy: the dialer must see the real destination to reject internal addresses.
				DialContext:           dialer.DialContext,
				ResponseHeaderTimeout: 5 * time.Second,
			},
			CheckRedirect: func(_ *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return http.ErrUseLastResponse
				}

				return nil
			},
		},
		api:       resty.New().SetBaseURL("https://api.github.com").SetTimeout(5 * time.Second),
		webOrigin: "https://",
	}
}

// requirePublicAddress runs after DNS resolution, so a hostname cannot point the server at itself
// or at its network, including through a redirect or a changing DNS answer.
func requirePublicAddress(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}

	ip, err := netip.ParseAddr(host)
	if err != nil || !isPublic(ip) {
		return errBlockedAddress
	}

	return nil
}

var carrierGradeNAT = netip.MustParsePrefix("100.64.0.0/10")

func isPublic(ip netip.Addr) bool {
	ip = ip.Unmap()

	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !carrierGradeNAT.Contains(ip)
}

// Lookup returns what the site or repo says about itself. Callers treat an error as "not found".
func (c *Client) Lookup(ctx context.Context, host, githubRepo string) (domain.BrandLookup, error) {
	if githubRepo != "" {
		return c.lookupRepo(ctx, githubRepo)
	}

	return c.lookupWebsite(ctx, host)
}

func (c *Client) lookupRepo(ctx context.Context, repo string) (domain.BrandLookup, error) {
	parsed, err := url.Parse(repo)
	if err != nil {
		return domain.BrandLookup{}, err
	}

	var result struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	resp, err := c.api.R().SetContext(ctx).SetHeader("Accept", "application/vnd.github+json").
		SetResult(&result).Get("/repos" + parsed.Path)
	if err != nil {
		return domain.BrandLookup{}, err
	}
	if !resp.IsSuccess() {
		return domain.BrandLookup{}, fmt.Errorf("github returned %s", resp.Status())
	}

	return domain.BrandLookup{
		Found:       result.Name != "",
		Name:        titleCase(result.Name),
		Description: limit(result.Description, maxDescriptionLen),
		Source:      domain.BrandLookupGitHub,
	}, nil
}

func (c *Client) lookupWebsite(ctx context.Context, host string) (domain.BrandLookup, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.webOrigin+host, nil)
	if err != nil {
		return domain.BrandLookup{}, err
	}
	req.Header.Set("Accept", "text/html")
	req.Header.Set("User-Agent", "Koi/1.0 (+https://koi.dev)")

	resp, err := c.web.Do(req)
	if err != nil {
		return domain.BrandLookup{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK || !strings.Contains(resp.Header.Get("Content-Type"), "html") {
		return domain.BrandLookup{}, fmt.Errorf("site returned %s", resp.Status)
	}

	page, err := io.ReadAll(io.LimitReader(resp.Body, maxPageBytes))
	if err != nil {
		return domain.BrandLookup{}, err
	}

	lookup := parsePage(string(page))
	lookup.Source = domain.BrandLookupWebsite

	return lookup, nil
}

var (
	metaTag       = regexp.MustCompile(`(?is)<meta\s[^>]*>`)
	metaAttribute = regexp.MustCompile(`(?is)\b(name|property|content)\s*=\s*("[^"]*"|'[^']*')`)
	titleTag      = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	titleSplit    = regexp.MustCompile(`\s+[|–—·•-]\s+|:\s+`)
)

func parsePage(page string) domain.BrandLookup {
	meta := map[string]string{}

	for _, tag := range metaTag.FindAllString(page, -1) {
		var key, content string

		for _, attribute := range metaAttribute.FindAllStringSubmatch(tag, -1) {
			value := html.UnescapeString(strings.Trim(attribute[2], `"'`))
			if strings.EqualFold(attribute[1], "content") {
				content = value
			} else {
				key = strings.ToLower(value)
			}
		}

		if key != "" && meta[key] == "" {
			meta[key] = strings.Join(strings.Fields(content), " ")
		}
	}

	name := meta["og:site_name"]
	if name == "" {
		name = meta["application-name"]
	}
	if name == "" {
		if match := titleTag.FindStringSubmatch(page); match != nil {
			title := strings.Join(strings.Fields(html.UnescapeString(match[1])), " ")
			name = strings.TrimSpace(titleSplit.Split(title, 2)[0])
		}
	}

	// A long "name" is a marketing headline. The brand name decides how answers are scored, so
	// the form keeps its own suggestion instead of trusting one.
	if len(name) > maxNameLength {
		name = ""
	}

	description := meta["description"]
	if description == "" {
		description = meta["og:description"]
	}

	return domain.BrandLookup{
		Found:       name != "",
		Name:        name,
		Description: limit(description, maxDescriptionLen),
	}
}

func titleCase(name string) string {
	if name == "" || name != strings.ToLower(name) {
		return name
	}

	return strings.ToUpper(name[:1]) + name[1:]
}

func limit(value string, length int) string {
	value = strings.TrimSpace(value)
	if len(value) <= length {
		return value
	}

	return strings.ToValidUTF8(value[:length], "")
}
