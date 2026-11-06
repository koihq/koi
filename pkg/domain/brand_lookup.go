package domain

import (
	"context"
	"log/slog"
)

const (
	BrandLookupWebsite = "website"
	BrandLookupGitHub  = "github"
)

// BrandLookup is what a brand's own website or repository says about it. It only suggests values
// for the setup form, so anything it gets wrong stays editable.
type BrandLookup struct {
	Found       bool   `json:"found"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
}

type BrandLooker interface {
	Lookup(ctx context.Context, host, githubRepo string) (BrandLookup, error)
}

// LookupBrand validates the identifiers first, so only a real hostname or repository is fetched.
// Setup works without an answer, so a failed lookup is logged and reported as not found.
func LookupBrand(ctx context.Context, looker BrandLooker, host, githubRepo string) (BrandLookup, error) {
	if (host == "") == (githubRepo == "") {
		return BrandLookup{}, Validation(FieldError{Field: "domain", Code: FieldInvalid})
	}
	if host != "" && !validHostname(host) {
		return BrandLookup{}, Validation(FieldError{Field: "domain", Code: FieldInvalid})
	}
	if githubRepo != "" && !validGitHubRepo(githubRepo) {
		return BrandLookup{}, Validation(FieldError{Field: "github_repo", Code: FieldInvalid})
	}
	if looker == nil {
		return BrandLookup{}, nil
	}

	lookup, err := looker.Lookup(ctx, host, githubRepo)
	if err != nil {
		slog.Debug("brand lookup failed", "err", err)

		return BrandLookup{}, nil
	}

	return lookup, nil
}
