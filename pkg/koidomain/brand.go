package koidomain

import (
	"context"
	"net"
	"net/url"
	"slices"
	"sort"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

type Brand struct {
	BaseEntity
	UserID      uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	Name        string    `json:"name" gorm:"not null"`
	Domain      string    `json:"domain"`
	GitHubRepo  string    `json:"github_repo" gorm:"column:github_repo"`
	Country     string    `json:"country" gorm:"not null;type:varchar(2)"`
	Description string    `json:"description"`
}

func (Brand) TableName() string { return "brands" }

func (b *Brand) Normalize() {
	b.Name = strings.TrimSpace(b.Name)
	b.Domain = strings.TrimSpace(b.Domain)
	b.GitHubRepo = strings.TrimSpace(b.GitHubRepo)
	b.Country = strings.ToUpper(strings.TrimSpace(b.Country))
	b.Description = strings.TrimSpace(b.Description)
}

func (b Brand) Validate() error {
	if b.UserID == uuid.Nil {
		return Validation(FieldError{Field: "user_id", Code: FieldInvalid})
	}
	if strings.TrimSpace(b.Name) == "" {
		return Validation(FieldError{Field: "name", Code: FieldInvalid})
	}
	if b.Domain == "" && b.GitHubRepo == "" {
		return Validation(
			FieldError{Field: "domain", Code: FieldInvalid},
			FieldError{Field: "github_repo", Code: FieldInvalid},
		)
	}
	if b.Domain != "" && !validHostname(b.Domain) {
		return Validation(FieldError{Field: "domain", Code: FieldInvalid})
	}
	if b.GitHubRepo != "" && !validGitHubRepo(b.GitHubRepo) {
		return Validation(FieldError{Field: "github_repo", Code: FieldInvalid})
	}
	if !supportedCountry(b.Country) {
		return Validation(FieldError{Field: "country", Code: FieldInvalid})
	}
	return nil
}

func supportedCountry(code string) bool {
	return slices.Contains(cloroCountries, code)
}

// SupportedCountries returns the Cloro snapshot used by Brand.Validate.
func SupportedCountries() []string {
	return slices.Clone(cloroCountries)
}

type CountryOption struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// SearchSupportedCountries returns all Cloro-supported countries by English
// name when query is empty. Searches return up to 20 countries ordered by
// exact code, code prefix, then name prefix.
func SearchSupportedCountries(query string) []CountryOption {
	query = strings.ToLower(strings.TrimSpace(query))

	type match struct {
		CountryOption
		rank int
	}
	matches := make([]match, 0, len(cloroCountries))
	namer := display.English.Regions()
	nameOrder := collate.New(language.English)
	for _, code := range cloroCountries {
		region, err := language.ParseRegion(code)
		if err != nil {
			continue
		}
		name := namer.Name(region)
		lowerCode, lowerName := strings.ToLower(code), strings.ToLower(name)
		rank := -1
		switch {
		case query == "":
			rank = 0
		case lowerCode == query:
			rank = 0
		case strings.HasPrefix(lowerCode, query):
			rank = 1
		case strings.HasPrefix(lowerName, query):
			rank = 2
		}
		if rank >= 0 {
			matches = append(matches, match{
				CountryOption: CountryOption{Code: code, Name: name},
				rank:          rank,
			})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].rank != matches[j].rank {
			return matches[i].rank < matches[j].rank
		}
		if order := nameOrder.CompareString(matches[i].Name, matches[j].Name); order != 0 {
			return order < 0
		}
		return matches[i].Code < matches[j].Code
	})
	if query != "" && len(matches) > 20 {
		matches = matches[:20]
	}
	result := make([]CountryOption, len(matches))
	for i := range matches {
		result[i] = matches[i].CountryOption
	}
	return result
}

func validHostname(host string) bool {
	if len(host) == 0 || len(host) > 253 || net.ParseIP(host) != nil {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	return true
}

func validGitHubRepo(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Scheme, "https") || !strings.EqualFold(u.Hostname(), "github.com") || u.Port() != "" || u.User != nil || u.ForceQuery || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	if strings.Contains(u.EscapedPath(), "%") {
		return false
	}
	path := strings.TrimSuffix(u.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 3 || parts[0] != "" || parts[1] == "" || parts[2] == "" {
		return false
	}
	if len(parts[1]) > 39 || parts[1][0] == '-' || parts[1][len(parts[1])-1] == '-' || parts[2] == "." || parts[2] == ".." {
		return false
	}
	for i, part := range parts[1:] {
		for _, c := range part {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || i == 1 && (c == '_' || c == '.')) {
				return false
			}
		}
	}
	return true
}

type BrandRepository interface {
	Create(context.Context, *Brand) error
	CountByUserID(context.Context, uuid.UUID) (int64, error)
	ListByUserID(context.Context, uuid.UUID, int, int) ([]Brand, error)
	GetByIDAndUserID(context.Context, uuid.UUID, uuid.UUID) (*Brand, error)
	UpdateByIDAndUserID(context.Context, uuid.UUID, *Brand) error
}

type BrandPage struct {
	Items       []Brand `json:"items"`
	TotalItems  int64   `json:"totalItems"`
	TotalPages  int64   `json:"totalPages"`
	CurrentPage int     `json:"currentPage"`
	HasNext     bool    `json:"hasNext"`
	HasPrev     bool    `json:"hasPrev"`
}

type BrandParams struct {
	Name        string
	Domain      string
	GitHubRepo  string
	Country     string
	Description string
}

type BrandCommander interface {
	Create(ctx context.Context, userID uuid.UUID, p BrandParams) (*Brand, error)
	Update(ctx context.Context, userID, id uuid.UUID, p BrandPatch) (*Brand, error)
}

type BrandQuerier interface {
	List(ctx context.Context, userID uuid.UUID, page, pageSize int) (*BrandPage, error)
	Get(ctx context.Context, userID, id uuid.UUID) (*Brand, error)
}

type BrandPatch struct {
	Name        *string
	Domain      *string
	GitHubRepo  *string
	Country     *string
	Description *string
}

type brandCommander struct{ store Store }

func NewBrandCommander(store Store) BrandCommander { return &brandCommander{store: store} }

func (c *brandCommander) Create(ctx context.Context, userID uuid.UUID, p BrandParams) (*Brand, error) {
	brand := &Brand{
		UserID: userID, Name: p.Name, Domain: p.Domain, GitHubRepo: p.GitHubRepo,
		Country: p.Country, Description: p.Description,
	}
	brand.Normalize()
	if err := brand.Validate(); err != nil {
		return nil, err
	}
	err := c.store.Atomic(ctx, func(s Store) error {
		if err := s.UserRepo().LockForUpdate(ctx, userID); err != nil {
			return err
		}
		count, err := s.BrandRepo().CountByUserID(ctx, userID)
		if err != nil {
			return err
		}
		if count != 0 {
			return ErrBrandLimitReached
		}
		return s.BrandRepo().Create(ctx, brand)
	})
	if err != nil {
		return nil, err
	}
	return brand, nil
}

func (c *brandCommander) Update(ctx context.Context, userID, id uuid.UUID, p BrandPatch) (*Brand, error) {
	brand, err := c.store.BrandRepo().GetByIDAndUserID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if p.Name != nil {
		brand.Name = *p.Name
	}
	if p.Domain != nil {
		brand.Domain = *p.Domain
	}
	if p.GitHubRepo != nil {
		brand.GitHubRepo = *p.GitHubRepo
	}
	if p.Country != nil {
		brand.Country = *p.Country
	}
	if p.Description != nil {
		brand.Description = *p.Description
	}
	brand.Normalize()
	if err := brand.Validate(); err != nil {
		return nil, err
	}
	if err := c.store.BrandRepo().UpdateByIDAndUserID(ctx, userID, brand); err != nil {
		return nil, err
	}
	return brand, nil
}

type brandQuerier struct{ store Store }

func NewBrandQuerier(store Store) BrandQuerier { return &brandQuerier{store: store} }

func (q *brandQuerier) List(ctx context.Context, userID uuid.UUID, page, pageSize int) (*BrandPage, error) {
	total, err := q.store.BrandRepo().CountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := &BrandPage{
		Items: make([]Brand, 0), TotalItems: total,
		CurrentPage: page, HasPrev: page > 1,
	}
	if total > 0 {
		result.TotalPages = (total-1)/int64(pageSize) + 1
	}
	result.HasNext = int64(page) < result.TotalPages
	if int64(page) > result.TotalPages {
		return result, nil
	}
	result.Items, err = q.store.BrandRepo().ListByUserID(ctx, userID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (q *brandQuerier) Get(ctx context.Context, userID, id uuid.UUID) (*Brand, error) {
	return q.store.BrandRepo().GetByIDAndUserID(ctx, id, userID)
}
