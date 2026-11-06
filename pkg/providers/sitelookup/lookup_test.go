package sitelookup

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/domain"
)

func TestParsePagePrefersTheSiteName(t *testing.T) {
	for _, test := range []struct {
		name, page, want string
	}{
		{"site name", `<meta property="og:site_name" content="Plausible"><title>Other | Thing</title>`, "Plausible"},
		{"reversed attributes", `<meta content='Acme &amp; Co' property='og:site_name'>`, "Acme & Co"},
		{"application name", `<meta name="application-name" content="Linear"><title>Plan</title>`, "Linear"},
		{"first title part", `<title>
			Deja | Remember everything</title>`, "Deja"},
		{"colon title", `<title>Fathom: simple analytics</title>`, "Fathom"},
		{"keeps a hyphenated name", `<title>Hugging-Face</title>`, "Hugging-Face"},
		{"rejects a headline", `<title>The simplest way to understand everything about your customers</title>`, ""},
		{"nothing", `<html></html>`, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			lookup := parsePage(test.page)
			require.Equal(t, test.want, lookup.Name)
			require.Equal(t, test.want != "", lookup.Found)
		})
	}
}

func TestParsePageReadsAndLimitsTheDescription(t *testing.T) {
	lookup := parsePage(`<meta name="description" content="  A  tool ">`)
	require.Equal(t, "A tool", lookup.Description)

	long := parsePage(`<meta name="description" content="` + strings.Repeat("a", 500) + `">`)
	require.Len(t, long.Description, maxDescriptionLen)
}

func TestOnlyPublicAddressesAreReachable(t *testing.T) {
	for address, public := range map[string]bool{
		"93.184.216.34":      true,
		"2606:4700::1111":    true,
		"127.0.0.1":          false,
		"10.0.0.1":           false,
		"172.16.0.1":         false,
		"192.168.1.1":        false,
		"169.254.169.254":    false,
		"100.64.0.1":         false,
		"0.0.0.0":            false,
		"::1":                false,
		"fe80::1":            false,
		"fd00::1":            false,
		"::ffff:127.0.0.1":   false,
		"::ffff:192.168.0.1": false,
	} {
		require.Equal(t, public, isPublic(netip.MustParseAddr(address)), address)
	}
}

func TestLookupWebsiteRefusesTheLocalNetwork(t *testing.T) {
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits++ }))
	defer server.Close()

	client := New()
	client.webOrigin = "http://"

	_, err := client.Lookup(t.Context(), strings.TrimPrefix(server.URL, "http://"), "")
	require.ErrorIs(t, err, errBlockedAddress)
	require.Zero(t, hits)
}

func TestLookupReposReadsGitHub(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/koihq/koi", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"koi","description":"AEO for founders"}`))
	}))
	defer server.Close()

	client := New()
	client.api.SetBaseURL(server.URL)

	lookup, err := client.Lookup(t.Context(), "", "https://github.com/koihq/koi")
	require.NoError(t, err)
	require.Equal(t, domain.BrandLookup{
		Found:       true,
		Name:        "Koi",
		Description: "AEO for founders",
		Source:      domain.BrandLookupGitHub,
	}, lookup)
}
