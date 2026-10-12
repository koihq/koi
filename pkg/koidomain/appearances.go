package koidomain

import (
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/text/cases"
)

type NamedEntity struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type AppearanceEvidence struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type AppearanceObservation struct {
	Key      string               `json:"key"`
	Name     string               `json:"name"`
	URLs     []string             `json:"urls"`
	Evidence []AppearanceEvidence `json:"evidence"`
}

type CompetitorAppearance struct {
	CompetitorID *uuid.UUID           `json:"competitor_id"`
	Name         string               `json:"name"`
	Confirmed    bool                 `json:"confirmed"`
	Evidence     []AppearanceEvidence `json:"evidence"`
}

// CompetitorObservation retains dismissal identities after the competitor itself is deleted.
// Counts advance only inside the first-save transaction for an independent visibility cell.
type CompetitorObservation struct {
	BaseEntity
	BrandID   uuid.UUID  `json:"-" gorm:"type:uuid;not null;uniqueIndex:idx_competitor_observations_key,priority:1"`
	Key       string     `json:"-" gorm:"not null;uniqueIndex:idx_competitor_observations_key,priority:2"`
	Count     int64      `json:"-" gorm:"not null"`
	Dismissed bool       `json:"-" gorm:"not null;default:false"`
	Identity  Competitor `json:"-" gorm:"type:jsonb;serializer:json;not null"`
	Brand     *Brand     `json:"-" gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
}

func (CompetitorObservation) TableName() string { return "competitor_observations" }

func AppearanceNameKey(name string) string {
	return cases.Fold().String(normalizeWhitespace(name))
}

func normalizeNames(names []string) []string {
	result := []string{}
	seen := map[string]bool{}
	for _, name := range names {
		name = normalizeWhitespace(name)
		key := AppearanceNameKey(name)
		if key != "" && !seen[key] {
			seen[key] = true
			result = append(result, name)
		}
	}
	return result
}

// NormalizeIdentityURL keeps website origins and exact GitHub repository roots.
func NormalizeIdentityURL(raw string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.User != nil || u.Port() != "" || (u.Scheme != "https" && u.Scheme != "http") {
		return "", false
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	if !validHostname(host) {
		return "", false
	}
	u.Host, u.RawQuery, u.Fragment, u.ForceQuery = host, "", "", false
	if host == "github.com" && strings.Trim(u.Path, "/") != "" {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) != 2 || u.RawPath != "" {
			return "", false
		}
		u.Scheme = "https"
		u.Path = "/" + strings.ToLower(strings.Join(parts, "/"))
		if !validGitHubRepo(u.String()) {
			return "", false
		}
	} else {
		u.Path, u.RawPath = "", ""
	}
	return u.String(), true
}

func normalizeIdentityURLs(values []string) []string {
	result := []string{}
	seen := map[string]bool{}
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if normalized, ok := NormalizeIdentityURL(value); ok {
			value = normalized
		}
		if value != "" && !seen[identityURLKey(value)] {
			seen[identityURLKey(value)] = true
			result = append(result, value)
		}
	}
	return result
}

func identityURLKey(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return strings.ToLower(u.Host + u.Path)
}

func (c Competitor) IdentityURLs() []string {
	result := slices.Clone(c.AdditionalURLs)
	if c.Domain != "" {
		result = append(result, "https://"+strings.TrimPrefix(c.Domain, "www."))
	}
	if c.GitHubRepo != "" {
		result = append(result, c.GitHubRepo)
	}
	return normalizeIdentityURLs(result)
}

func (c Competitor) Names() []string {
	return normalizeNames(append([]string{c.Name}, c.Aliases...))
}

func (c Competitor) SharesIdentityURL(other Competitor) bool {
	for _, a := range c.IdentityURLs() {
		for _, b := range other.IdentityURLs() {
			if identityURLKey(a) == identityURLKey(b) {
				return true
			}
		}
	}
	return false
}

func identityURLScore(c Competitor, raw string) int {
	source, err := url.Parse(raw)
	if err != nil || source.User != nil || (source.Scheme != "http" && source.Scheme != "https") {
		return 0
	}
	host := strings.TrimPrefix(strings.ToLower(source.Hostname()), "www.")
	best := 0
	for _, value := range c.IdentityURLs() {
		identity, err := url.Parse(value)
		if err != nil {
			continue
		}
		if identity.Host == "github.com" && identity.Path != "" {
			path := strings.ToLower(source.Path)
			if host == identity.Host && (path == identity.Path || strings.HasPrefix(path, identity.Path+"/")) {
				best = max(best, 1000+len(identity.Path))
			}
		} else if host == identity.Host || strings.HasSuffix(host, "."+identity.Host) {
			best = max(best, len(identity.Host))
		}
	}
	return best
}

// ResolveObservation returns the most specific URL identity, falling back to exact names.
// Ambiguous names and URL ties stay unresolved instead of attaching to an arbitrary row.
func ResolveObservation(observation AppearanceObservation, competitors []Competitor) (int, bool) {
	best, score, tied := -1, 0, false
	for i, competitor := range competitors {
		value := 0
		for _, link := range observation.URLs {
			value = max(value, identityURLScore(competitor, link))
		}
		if value > score {
			best, score, tied = i, value, false
		} else if value > 0 && value == score {
			tied = true
		}
	}
	if score > 0 {
		if tied {
			return -1, true
		}
		return best, false
	}
	for i, competitor := range competitors {
		for _, name := range competitor.Names() {
			if AppearanceNameKey(name) == AppearanceNameKey(observation.Name) {
				if best != -1 && best != i {
					return -1, true
				}
				best = i
			}
		}
	}
	return best, false
}

func ObservationIsBrand(observation AppearanceObservation, brand BrandSnapshot) bool {
	index, _ := ResolveObservation(observation, []Competitor{
		{
			Name:       brand.Name,
			Domain:     brand.Domain,
			GitHubRepo: brand.GitHubRepo,
		},
	})
	return index == 0
}

var leadingProductLink = regexp.MustCompile(`^(?:#{1,6}\s+|[-+*]\s+|[0-9]+[.)]\s+|\|\s*)(?:\*\*)?\[([^\]\n]+)\]\((https?://[^\s)]+)\)`)

// ExtractObservations intentionally accepts only entities and leading product homepage links.
// ponytail: shallow Markdown parsing misses complex links; use an AST if fixtures require it.
func ExtractObservations(result VisibilityResult) []AppearanceObservation {
	observations := []AppearanceObservation{}
	positions := map[string]int{}
	add := func(name, link, kind string) {
		name = normalizeWhitespace(name)
		key := AppearanceNameKey(name)
		if key == "" || len(name) > 255 {
			return
		}
		position, exists := positions[key]
		if !exists {
			position = len(observations)
			positions[key] = position
			observations = append(observations, AppearanceObservation{
				Key:      key,
				Name:     name,
				URLs:     []string{},
				Evidence: []AppearanceEvidence{},
			})
		}
		item := &observations[position]
		item.Evidence = appendEvidence(item.Evidence, AppearanceEvidence{Kind: kind, Value: name})
		if link != "" {
			item.URLs = normalizeIdentityURLs(append(item.URLs, link))
			item.Evidence = appendEvidence(item.Evidence, AppearanceEvidence{Kind: "link", Value: link})
		}
	}
	for _, entity := range result.NamedEntities {
		if slices.Contains([]string{"product", "software", "brand", "company", "organization"}, strings.ToLower(entity.Type)) {
			add(entity.Name, "", "entity")
		}
	}
	markdown := result.AnswerMarkdown
	if markdown == "" {
		markdown = result.RawAnswer
	}
	for _, line := range strings.Split(markdown, "\n") {
		match := leadingProductLink.FindStringSubmatch(strings.TrimSpace(line))
		if len(match) == 0 {
			continue
		}
		u, err := url.Parse(match[2])
		if err != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			continue
		}
		// Deep website pages are usually articles or citations, not product identities.
		if strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.") != "github.com" && strings.Trim(u.Path, "/") != "" {
			continue
		}
		link, ok := NormalizeIdentityURL(match[2])
		if ok && strings.Trim(match[1], "0123456789 []") != "" && !strings.HasPrefix(strings.ToLower(match[1]), "http") {
			add(match[1], link, "mention")
		}
	}
	return observations
}

func appendEvidence(items []AppearanceEvidence, evidence AppearanceEvidence) []AppearanceEvidence {
	if !slices.Contains(items, evidence) {
		items = append(items, evidence)
	}
	return items
}

// MatchAppearances derives current links without changing the original answer or outcome.
func MatchAppearances(result VisibilityResult, competitors []Competitor, brand BrandSnapshot) []CompetitorAppearance {
	appearances := []CompetitorAppearance{}
	if result.Outcome == nil || *result.Outcome == OutcomeError {
		return appearances
	}
	positions := map[uuid.UUID]int{}
	add := func(index int, evidence []AppearanceEvidence) {
		competitor := competitors[index]
		position, exists := positions[competitor.ID]
		if !exists {
			position = len(appearances)
			positions[competitor.ID] = position
			id := competitor.ID
			appearances = append(appearances, CompetitorAppearance{
				CompetitorID: &id,
				Name:         competitor.Name,
				Confirmed:    competitor.Confirmed,
				Evidence:     []AppearanceEvidence{},
			})
		}
		for _, item := range evidence {
			appearances[position].Evidence = appendEvidence(appearances[position].Evidence, item)
		}
	}
	for i, competitor := range competitors {
		for _, name := range competitor.Names() {
			if ScoreVisibility(BrandSnapshot{Name: name}, result.RawAnswer, nil) == OutcomeMentioned {
				index, _ := ResolveObservation(AppearanceObservation{Name: name}, competitors)
				if index == i {
					add(i, []AppearanceEvidence{{Kind: "mention", Value: name}})
				}
			}
		}
	}
	for _, source := range result.Sources {
		index, _ := ResolveObservation(AppearanceObservation{
			URLs: []string{source},
		}, competitors)
		if index >= 0 {
			add(index, []AppearanceEvidence{{Kind: "citation", Value: source}})
		}
	}
	for _, observation := range result.Observations {
		if ObservationIsBrand(observation, brand) {
			continue
		}
		index, _ := ResolveObservation(observation, competitors)
		if index >= 0 {
			add(index, observation.Evidence)
		} else {
			appearances = append(appearances, CompetitorAppearance{Name: observation.Name, Evidence: slices.Clone(observation.Evidence)})
		}
	}
	return appearances
}
