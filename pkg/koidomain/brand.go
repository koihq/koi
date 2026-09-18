package koidomain

import (
	"context"
	"net"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/text/language"
)

type Brand struct {
	BaseEntity
	UserID      uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	Name        string    `json:"name" gorm:"not null"`
	Domain      string    `json:"domain"`
	GitHubRepo  string    `json:"github_repo" gorm:"column:github_repo"`
	Country     string    `json:"country" gorm:"not null;type:varchar(2)"`
	ProductLine string    `json:"product_line"`
}

func (Brand) TableName() string { return "brands" }

func (b *Brand) Normalize() {
	b.Name = strings.TrimSpace(b.Name)
	b.Domain = strings.TrimSpace(b.Domain)
	b.GitHubRepo = strings.TrimSpace(b.GitHubRepo)
	b.Country = strings.ToUpper(strings.TrimSpace(b.Country))
	b.ProductLine = strings.TrimSpace(b.ProductLine)
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
	region, err := language.ParseRegion(b.Country)
	if len(b.Country) != 2 || err != nil || !region.IsCountry() || region.IsPrivateUse() || region.Canonicalize().String() != b.Country {
		return Validation(FieldError{Field: "country", Code: FieldInvalid})
	}
	return nil
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
	ListByUserID(context.Context, uuid.UUID) ([]Brand, error)
}

type BrandParams struct {
	Name        string
	Domain      string
	GitHubRepo  string
	Country     string
	ProductLine string
}

type BrandCommander interface {
	Create(ctx context.Context, userID uuid.UUID, p BrandParams) (*Brand, error)
}

type brandCommander struct{ store Store }

func NewBrandCommander(store Store) BrandCommander { return &brandCommander{store: store} }

func (c *brandCommander) Create(ctx context.Context, userID uuid.UUID, p BrandParams) (*Brand, error) {
	brand := &Brand{
		UserID: userID, Name: p.Name, Domain: p.Domain, GitHubRepo: p.GitHubRepo,
		Country: p.Country, ProductLine: p.ProductLine,
	}
	brand.Normalize()
	if err := brand.Validate(); err != nil {
		return nil, err
	}
	err := c.store.Atomic(ctx, func(s Store) error {
		if err := s.UserRepo().LockForUpdate(ctx, userID); err != nil {
			return err
		}
		brands, err := s.BrandRepo().ListByUserID(ctx, userID)
		if err != nil {
			return err
		}
		if len(brands) != 0 {
			return ErrBrandLimitReached
		}
		return s.BrandRepo().Create(ctx, brand)
	})
	if err != nil {
		return nil, err
	}
	return brand, nil
}
