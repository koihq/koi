package koidomain

import (
	"context"
	"net"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/koihq/koi/pkg/koiauth"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

type Brand struct {
	PromptsReviewedAt     *time.Time `json:"-"`
	CompetitorsReviewedAt *time.Time `json:"-"`
	VisibilityStartedAt   *time.Time `json:"-"`
	NextVisibilityAt      *time.Time `json:"-"`
	BaseEntity
	UserID      uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	Name        string    `json:"name" gorm:"not null"`
	Domain      string    `json:"domain"`
	GitHubRepo  string    `json:"github_repo" gorm:"column:github_repo"`
	Country     string    `json:"country" gorm:"not null;type:varchar(2)"`
	Description string    `json:"description"`
}

func (Brand) TableName() string { return "brands" }

func (b Brand) Snapshot() BrandSnapshot {
	return BrandSnapshot{
		Name:        b.Name,
		Domain:      b.Domain,
		GitHubRepo:  b.GitHubRepo,
		Country:     b.Country,
		Description: b.Description,
	}
}

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
	return slices.Contains(countryCodes, code)
}

func SupportedCountries() []string {
	return slices.Clone(countryCodes)
}

type CountryOption struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

var countryOptions = loadCountryOptions()

func loadCountryOptions() []CountryOption {
	namer := display.English.Regions()
	opts := make([]CountryOption, 0, len(countryCodes))

	for _, code := range countryCodes {
		name := code
		if region, err := language.ParseRegion(code); err == nil {
			if n := namer.Name(region); n != "" {
				name = n
			}
		}

		opts = append(opts, CountryOption{Code: code, Name: name})
	}

	nameOrder := collate.New(language.English)
	slices.SortFunc(opts, func(a, b CountryOption) int {
		if order := nameOrder.CompareString(a.Name, b.Name); order != 0 {
			return order
		}

		return strings.Compare(a.Code, b.Code)
	})

	return opts
}

// SearchSupportedCountries returns supported countries by English
// name when query is empty. Searches return up to 20 countries ordered by
// exact code, code prefix, then name prefix.
func SearchSupportedCountries(query string) []CountryOption {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return slices.Clone(countryOptions)
	}

	var exact, codes, names []CountryOption

	for _, c := range countryOptions {
		code, name := strings.ToLower(c.Code), strings.ToLower(c.Name)

		switch {
		case code == query:
			exact = append(exact, c)
		case strings.HasPrefix(code, query):
			codes = append(codes, c)
		case strings.HasPrefix(name, query):
			names = append(names, c)
		}
	}

	out := slices.Concat(exact, codes, names)
	if len(out) > 20 {
		out = out[:20]
	}

	return out
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
	BaseEntityRepository[Brand, uuid.UUID]
	LockForUpdate(context.Context, uuid.UUID) (*Brand, error)
	Count(context.Context) (int64, error)
	List(context.Context, PageParams) (*Page[Brand], error)
	Update(context.Context, *Brand) error
}

type BrandParams struct {
	Name        string
	Domain      string
	GitHubRepo  string
	Country     string
	Description string
}

type BrandCommander interface {
	Create(ctx context.Context, p BrandParams) (*Brand, error)
	Update(ctx context.Context, id uuid.UUID, p BrandPatch) (*Brand, error)
}

type BrandQuerier interface {
	List(ctx context.Context, params PageParams) (*Page[Brand], error)
	Get(ctx context.Context, id uuid.UUID) (*Brand, error)
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

func (c *brandCommander) Create(ctx context.Context, p BrandParams) (*Brand, error) {
	identity, ok := koiauth.FromContext(ctx)
	if !ok {
		return nil, ErrUnauthorized
	}

	brand := &Brand{
		UserID:      identity.UserID,
		Name:        p.Name,
		Domain:      p.Domain,
		GitHubRepo:  p.GitHubRepo,
		Country:     p.Country,
		Description: p.Description,
	}

	brand.Normalize()
	if err := brand.Validate(); err != nil {
		return nil, err
	}

	err := c.store.Atomic(ctx, func(s Store) error {
		if err := s.UserRepo().LockForUpdate(ctx, identity.UserID); err != nil {
			return err
		}

		count, err := s.BrandRepo().Count(ctx)
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

func (c *brandCommander) Update(ctx context.Context, id uuid.UUID, p BrandPatch) (*Brand, error) {
	brand, err := c.store.BrandRepo().Get(ctx, id)
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

	if err := c.store.BrandRepo().Update(ctx, brand); err != nil {
		return nil, err
	}

	return brand, nil
}

type brandQuerier struct{ store Store }

func NewBrandQuerier(store Store) BrandQuerier { return &brandQuerier{store: store} }

func (q *brandQuerier) List(ctx context.Context, params PageParams) (*Page[Brand], error) {
	return q.store.BrandRepo().List(ctx, params)
}

func (q *brandQuerier) Get(ctx context.Context, id uuid.UUID) (*Brand, error) {
	return q.store.BrandRepo().Get(ctx, id)
}
