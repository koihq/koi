package koiapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/koihq/koi/pkg/koidomain"
)

type mcpUUID string

func (id *mcpUUID) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	parsed, err := uuid.Parse(value)
	if err != nil || parsed == uuid.Nil {
		return fmt.Errorf("invalid UUID")
	}
	*id = mcpUUID(parsed.String())
	return nil
}
func (id mcpUUID) UUID() uuid.UUID { parsed, _ := uuid.Parse(string(id)); return parsed }

type mcpScopeKey struct{}

type mcpScope struct {
	BrandID mcpUUID `json:"brandId" jsonschema:"Brand UUID (required)"`
}
type mcpID struct {
	ID mcpUUID `json:"id" jsonschema:"Resource UUID"`
}
type mcpPage struct {
	BrandID  mcpUUID `json:"brandId"`
	Page     int     `json:"page,omitempty"`
	PageSize int     `json:"pageSize,omitempty"`
}
type mcpBrandCreate struct {
	Name        string `json:"name"`
	Domain      string `json:"domain"`
	GitHubRepo  string `json:"githubRepo,omitempty"`
	Country     string `json:"country"`
	Description string `json:"description,omitempty"`
}
type mcpBrandUpdate struct {
	ID          mcpUUID `json:"id"`
	Name        *string `json:"name,omitempty"`
	Domain      *string `json:"domain,omitempty"`
	GitHubRepo  *string `json:"githubRepo,omitempty"`
	Country     *string `json:"country,omitempty"`
	Description *string `json:"description,omitempty"`
}
type mcpPromptCreate struct {
	BrandID mcpUUID `json:"brandId"`
	Text    string  `json:"text"`
}
type mcpPromptUpdate struct {
	ID       mcpUUID `json:"id"`
	Text     *string `json:"text,omitempty"`
	Inactive *bool   `json:"inactive,omitempty"`
}
type mcpCompetitorCreate struct {
	BrandID        mcpUUID  `json:"brandId"`
	Name           string   `json:"name"`
	Domain         string   `json:"domain"`
	GitHubRepo     string   `json:"githubRepo,omitempty"`
	Aliases        []string `json:"aliases,omitempty"`
	AdditionalURLs []string `json:"additionalUrls,omitempty"`
}
type mcpCompetitorUpdate struct {
	ID             mcpUUID   `json:"id"`
	Name           *string   `json:"name,omitempty"`
	Domain         *string   `json:"domain,omitempty"`
	GitHubRepo     *string   `json:"githubRepo,omitempty"`
	Aliases        *[]string `json:"aliases,omitempty"`
	AdditionalURLs *[]string `json:"additionalUrls,omitempty"`
	Confirmed      *bool     `json:"confirmed,omitempty"`
}
type mcpGenerationCreate struct {
	BrandID        mcpUUID                  `json:"brandId"`
	Mode           koidomain.GenerationMode `json:"mode"`
	IdempotencyKey string                   `json:"idempotencyKey"`
}
type mcpAnalysis struct {
	BrandID mcpUUID              `json:"brandId"`
	Days    int                  `json:"days,omitempty"`
	Engines []koidomain.EngineID `json:"engines,omitempty"`
}
type mcpPromptHistory struct {
	BrandID  mcpUUID `json:"brandId"`
	PromptID mcpUUID `json:"promptId"`
	Page     int     `json:"page,omitempty"`
	PageSize int     `json:"pageSize,omitempty"`
}
type mcpQuote struct {
	Prompts int `json:"prompts"`
}
type mcpChange struct {
	Prompts        int    `json:"prompts"`
	IdempotencyKey string `json:"idempotencyKey"`
}
type mcpCheckoutConfirm struct {
	SessionID string `json:"sessionId"`
}
type mcpSearch struct {
	BrandID mcpUUID `json:"brandId"`
	Query   string  `json:"query"`
}

func mcpPageParams(page, size int) (koidomain.PageParams, error) {
	if page == 0 {
		page = 1
	}
	if size == 0 {
		size = 20
	}
	if page < 1 || size < 1 || size > 100 {
		return koidomain.PageParams{}, koidomain.Validation(koidomain.FieldError{Field: "page", Code: koidomain.FieldInvalid})
	}
	return koidomain.PageParams{Page: page, PageSize: size}, nil
}
func requireBrand(id uuid.UUID) error {
	if id == uuid.Nil {
		return koidomain.Validation(koidomain.FieldError{Field: "brandId", Code: koidomain.FieldInvalid})
	}
	return nil
}
func mcpAccess(ctx context.Context, billing *BillingHandler) error {
	if billing == nil {
		return nil
	}
	return koidomain.RequireProductAccess(ctx, billing.querier)
}
func addMCPTool[In any](server *mcp.Server, name, description string, readOnly bool, billing *BillingHandler, guard bool, fn func(context.Context, In) (any, error)) {
	mcp.AddTool(server, &mcp.Tool{Name: name, Description: description, Meta: mcp.Meta{"koi/scope": map[bool]string{true: "koi:read", false: "koi:write"}[readOnly]}, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: readOnly}}, func(ctx context.Context, _ *mcp.CallToolRequest, input In) (*mcp.CallToolResult, any, error) {
		scope, ok := ctx.Value(mcpScopeKey{}).(string)
		if !ok {
			return nil, nil, koidomain.ErrUnauthorized
		}
		required := "koi:write"
		if readOnly {
			required = "koi:read"
		}
		if !slices.Contains(strings.Fields(scope), required) {
			return nil, nil, koidomain.ErrUnauthorized
		}
		if guard {
			if err := mcpAccess(ctx, billing); err != nil {
				return nil, nil, err
			}
		}
		out, err := fn(ctx, input)
		return nil, out, err
	})
}

func newMCPHandler(store koidomain.Store, searcher koidomain.CompanySearcher, aiDrafting bool, billing *BillingHandler) http.Handler {
	server := mcp.NewServer(&mcp.Implementation{Name: "koi", Version: "1.0.0"}, nil)
	brands := koidomain.NewBrandQuerier(store)
	brandCommands := koidomain.NewBrandCommander(store)
	prompts := koidomain.NewPromptQuerier(store)
	promptCommands := koidomain.NewPromptCommander(store)
	competitors := koidomain.NewCompetitorQuerier(store, searcher)
	competitorCommands := koidomain.NewCompetitorCommander(store)
	generations := koidomain.NewGenerationQuerier(store)
	generationCommands := koidomain.NewGenerationCommander(store, aiDrafting)
	visibility := koidomain.NewVisibilityQuerier(store)
	analytics := koidomain.NewVisibilityAnalyticsQuerier(store)

	addMCPTool(server, "account_me", "Get your account and organization", true, billing, false, func(ctx context.Context, _ struct{}) (any, error) {
		v, e := koidomain.NewAuthQuerier(store).Me(ctx, uuid.Nil)
		if e != nil {
			return nil, e
		}
		return AuthToRes(v), nil
	})
	addMCPTool(server, "brands_countries", "Search supported country codes", true, billing, false, func(ctx context.Context, in struct {
		Query string `json:"query"`
	}) (any, error) {
		return koidomain.SearchSupportedCountries(in.Query), nil
	})
	addMCPTool(server, "brands_list", "List brands in your organization", true, billing, false, func(ctx context.Context, in struct {
		Page     int    `json:"page,omitempty"`
		PageSize int    `json:"pageSize,omitempty"`
		Query    string `json:"query,omitempty"`
	}) (any, error) {
		p, e := mcpPageParams(in.Page, in.PageSize)
		if e != nil {
			return nil, e
		}
		result, e := brands.List(ctx, koidomain.BrandListParams{Page: p, Query: in.Query})
		if e != nil {
			return nil, e
		}
		return mapPage(result, BrandToResponse), nil
	})
	addMCPTool(server, "brands_get", "Get a brand by UUID", true, billing, false, func(ctx context.Context, in mcpID) (any, error) {
		v, e := brands.Get(ctx, in.ID.UUID())
		if e != nil {
			return nil, e
		}
		return BrandToResponse(v), nil
	})
	addMCPTool(server, "brands_create", "Create a brand", false, billing, false, func(ctx context.Context, in mcpBrandCreate) (any, error) {
		v, e := brandCommands.Create(ctx, koidomain.BrandParams{Name: in.Name, Domain: in.Domain, GitHubRepo: in.GitHubRepo, Country: in.Country, Description: in.Description})
		if e != nil {
			return nil, e
		}
		return BrandToResponse(v), nil
	})
	addMCPTool(server, "brands_update", "Update a brand", false, billing, true, func(ctx context.Context, in mcpBrandUpdate) (any, error) {
		v, e := brandCommands.Update(ctx, in.ID.UUID(), koidomain.BrandPatch{Name: in.Name, Domain: in.Domain, GitHubRepo: in.GitHubRepo, Country: in.Country, Description: in.Description})
		if e != nil {
			return nil, e
		}
		return BrandToResponse(v), nil
	})
	addMCPTool(server, "prompts_list", "List prompts for a brand", true, billing, true, func(ctx context.Context, in mcpPage) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		p, e := mcpPageParams(in.Page, in.PageSize)
		if e != nil {
			return nil, e
		}
		v, e := prompts.List(ctx, in.BrandID.UUID(), koidomain.PromptListParams{Page: p})
		if e != nil {
			return nil, e
		}
		return mapPage(v, PromptToResponse), nil
	})
	addMCPTool(server, "prompts_get", "Get a prompt", true, billing, true, func(ctx context.Context, in mcpID) (any, error) {
		v, e := prompts.Get(ctx, in.ID.UUID())
		if e != nil {
			return nil, e
		}
		return PromptToResponse(v), nil
	})
	addMCPTool(server, "prompts_create", "Create a prompt for a brand", false, billing, true, func(ctx context.Context, in mcpPromptCreate) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		v, e := promptCommands.Create(ctx, in.BrandID.UUID(), koidomain.PromptParams{Text: in.Text})
		if e != nil {
			return nil, e
		}
		return PromptToResponse(v), nil
	})
	addMCPTool(server, "prompts_update", "Update a prompt", false, billing, true, func(ctx context.Context, in mcpPromptUpdate) (any, error) {
		v, e := promptCommands.Update(ctx, in.ID.UUID(), koidomain.PromptPatch{Text: in.Text, Inactive: in.Inactive})
		if e != nil {
			return nil, e
		}
		return PromptToResponse(v), nil
	})
	addMCPTool(server, "prompts_delete", "Delete a prompt", false, billing, true, func(ctx context.Context, in mcpID) (any, error) {
		return map[string]bool{"deleted": true}, promptCommands.Delete(ctx, in.ID.UUID())
	})
	addMCPTool(server, "prompts_confirm", "Confirm a prompt", false, billing, true, func(ctx context.Context, in mcpID) (any, error) {
		v, e := promptCommands.Confirm(ctx, in.ID.UUID())
		if e != nil {
			return nil, e
		}
		return PromptToResponse(v), nil
	})
	addMCPTool(server, "prompts_confirm_all", "Confirm all prompts for a brand", false, billing, true, func(ctx context.Context, in mcpScope) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		return map[string]bool{"confirmed": true}, promptCommands.ConfirmAll(ctx, in.BrandID.UUID())
	})
	addMCPTool(server, "competitors_list", "List competitors for a brand", true, billing, true, func(ctx context.Context, in mcpPage) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		p, e := mcpPageParams(in.Page, in.PageSize)
		if e != nil {
			return nil, e
		}
		v, e := competitors.List(ctx, in.BrandID.UUID(), p, "", nil)
		if e != nil {
			return nil, e
		}
		return mapPage(v, CompetitorToResponse), nil
	})
	addMCPTool(server, "competitors_get", "Get a competitor", true, billing, true, func(ctx context.Context, in mcpID) (any, error) {
		v, e := competitors.Get(ctx, in.ID.UUID())
		if e != nil {
			return nil, e
		}
		return CompetitorToResponse(v), nil
	})
	addMCPTool(server, "competitors_search", "Search suggested competitors", true, billing, true, func(ctx context.Context, in mcpSearch) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		return competitors.Search(ctx, in.BrandID.UUID(), in.Query)
	})
	addMCPTool(server, "competitors_create", "Create a competitor", false, billing, true, func(ctx context.Context, in mcpCompetitorCreate) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		v, e := competitorCommands.Create(ctx, in.BrandID.UUID(), koidomain.CompetitorParams{Name: in.Name, Domain: in.Domain, GitHubRepo: in.GitHubRepo, Aliases: in.Aliases, AdditionalURLs: in.AdditionalURLs})
		if e != nil {
			return nil, e
		}
		return CompetitorToResponse(v), nil
	})
	addMCPTool(server, "competitors_update", "Update a competitor", false, billing, true, func(ctx context.Context, in mcpCompetitorUpdate) (any, error) {
		v, e := competitorCommands.Patch(ctx, in.ID.UUID(), koidomain.CompetitorUpdate{Name: in.Name, Domain: in.Domain, GitHubRepo: in.GitHubRepo, Aliases: in.Aliases, AdditionalURLs: in.AdditionalURLs, Confirmed: in.Confirmed})
		if e != nil {
			return nil, e
		}
		return CompetitorToResponse(v), nil
	})
	addMCPTool(server, "competitors_delete", "Delete a competitor", false, billing, true, func(ctx context.Context, in mcpID) (any, error) {
		return map[string]bool{"deleted": true}, competitorCommands.Delete(ctx, in.ID.UUID())
	})
	addMCPTool(server, "competitors_confirm", "Confirm a competitor", false, billing, true, func(ctx context.Context, in mcpID) (any, error) {
		v, e := competitorCommands.Confirm(ctx, in.ID.UUID())
		if e != nil {
			return nil, e
		}
		return CompetitorToResponse(v), nil
	})
	addMCPTool(server, "competitors_confirm_all", "Confirm all competitors for a brand", false, billing, true, func(ctx context.Context, in mcpScope) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		return map[string]bool{"confirmed": true}, competitorCommands.ConfirmAll(ctx, in.BrandID.UUID())
	})
	addMCPTool(server, "generations_create", "Start an AI drafting job. Provide a stable idempotencyKey for retries", false, billing, true, func(ctx context.Context, in mcpGenerationCreate) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		if !validIdempotencyKey(in.IdempotencyKey) {
			return nil, koidomain.Validation(koidomain.FieldError{Field: "idempotencyKey", Code: koidomain.FieldInvalid})
		}
		v, replay, e := generationCommands.Create(ctx, in.BrandID.UUID(), in.Mode, &in.IdempotencyKey)
		if e != nil {
			return nil, e
		}
		return map[string]any{"id": v.ID, "state": v.State, "mode": v.Mode, "terminalReplay": replay}, nil
	})
	addMCPTool(server, "generations_get", "Get drafting job status", true, billing, true, func(ctx context.Context, in mcpID) (any, error) {
		v, e := generations.Get(ctx, in.ID.UUID())
		if e != nil {
			return nil, e
		}
		return publicGenerationStatus(v), nil
	})
	addMCPTool(server, "generations_list", "List drafting jobs", true, billing, true, func(ctx context.Context, in mcpPage) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		p, e := mcpPageParams(in.Page, in.PageSize)
		if e != nil {
			return nil, e
		}
		v, e := generations.List(ctx, in.BrandID.UUID(), p)
		if e != nil {
			return nil, e
		}
		return mapPage(v, publicGenerationStatus), nil
	})
	addMCPTool(server, "tracking_get", "Get live collection tracking state for a brand", true, billing, true, func(ctx context.Context, in mcpScope) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		v, e := visibility.Tracking(ctx, in.BrandID.UUID())
		if e != nil {
			return nil, e
		}
		return TrackingStatusToResponse(v), nil
	})
	addMCPTool(server, "runs_list", "List visibility runs", true, billing, true, func(ctx context.Context, in mcpPage) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		p, e := mcpPageParams(in.Page, in.PageSize)
		if e != nil {
			return nil, e
		}
		v, e := visibility.List(ctx, in.BrandID.UUID(), p)
		if e != nil {
			return nil, e
		}
		return mapPage(v, publicVisibilityStatus), nil
	})
	addMCPTool(server, "runs_get", "Get visibility run status", true, billing, true, func(ctx context.Context, in mcpID) (any, error) {
		v, e := visibility.Get(ctx, in.ID.UUID())
		if e != nil {
			return nil, e
		}
		return publicVisibilityStatus(v), nil
	})
	addMCPTool(server, "runs_results", "List collected answers and citations for a run", true, billing, true, func(ctx context.Context, in struct {
		ID       mcpUUID `json:"id"`
		Page     int     `json:"page,omitempty"`
		PageSize int     `json:"pageSize,omitempty"`
	}) (any, error) {
		p, e := mcpPageParams(in.Page, in.PageSize)
		if e != nil {
			return nil, e
		}
		v, e := visibility.ListResults(ctx, in.ID.UUID(), p)
		if e != nil {
			return nil, e
		}
		return mapPage(v, VisibilityResultToResponse), nil
	})
	addMCPTool(server, "analysis_summary", "Get visibility analysis for a brand", true, billing, true, func(ctx context.Context, in mcpAnalysis) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		if in.Days == 0 {
			in.Days = 30
		}
		if in.Days != 7 && in.Days != 30 && in.Days != 90 {
			return nil, fmt.Errorf("days must be 7, 30 or 90")
		}
		return analytics.Summary(ctx, in.BrandID.UUID(), in.Days, in.Engines)
	})
	addMCPTool(server, "prompt_visibility", "Get visibility analysis for a historical prompt", true, billing, true, func(ctx context.Context, in mcpPromptHistory) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		if _, e := visibility.ListPromptRuns(ctx, in.BrandID.UUID(), in.PromptID.UUID(), koidomain.PageParams{Page: 1, PageSize: 1}); e != nil {
			return nil, e
		}
		byPrompt, _, e := analytics.Aggregate(ctx, in.BrandID.UUID(), []uuid.UUID{in.PromptID.UUID()}, 30, nil)
		if e != nil {
			return nil, e
		}
		return byPrompt[in.PromptID.UUID()], nil
	})
	addMCPTool(server, "prompt_history_list", "List historical prompts with results", true, billing, true, func(ctx context.Context, in mcpPage) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		p, e := mcpPageParams(in.Page, in.PageSize)
		if e != nil {
			return nil, e
		}
		v, e := visibility.ListHistoricalPrompts(ctx, in.BrandID.UUID(), koidomain.PromptListParams{Page: p})
		if e != nil {
			return nil, e
		}
		return mapPage(v, HistoricalPromptToResponse), nil
	})
	addMCPTool(server, "prompt_history_runs", "List runs for a historical prompt", true, billing, true, func(ctx context.Context, in mcpPromptHistory) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		p, e := mcpPageParams(in.Page, in.PageSize)
		if e != nil {
			return nil, e
		}
		history, e := visibility.ListPromptRuns(ctx, in.BrandID.UUID(), in.PromptID.UUID(), p)
		if e != nil {
			return nil, e
		}
		items := make([]promptHistoryRunRes, 0, len(history.Items))
		for i := range history.Items {
			items = append(items, promptHistoryRunRes{Run: publicVisibilityStatus(&history.Items[i].Run), Results: mapItems(history.Items[i].Results, VisibilityResultToResponse)})
		}
		return koidomain.Page[promptHistoryRunRes]{Items: items, TotalItems: history.TotalItems, TotalPages: history.TotalPages, CurrentPage: history.CurrentPage, HasNext: history.HasNext, HasPrev: history.HasPrev}, nil
	})
	if billing != nil {
		addMCPTool(server, "billing_catalog", "Get subscription catalog", true, billing, false, func(ctx context.Context, _ struct{}) (any, error) {
			v, e := billing.querier.Catalog(ctx)
			if e != nil {
				return nil, e
			}
			return BillingCatalogToResponse(&v), nil
		})
		addMCPTool(server, "billing_quote", "Quote a prompt allowance", true, billing, false, func(ctx context.Context, in mcpQuote) (any, error) {
			v, e := billing.querier.Quote(ctx, in.Prompts)
			if e != nil {
				return nil, e
			}
			return BillingQuoteToResponse(&v), nil
		})
		addMCPTool(server, "billing_subscription", "Get reconciled subscription state", true, billing, false, func(ctx context.Context, _ struct{}) (any, error) {
			v, e := billing.querier.Subscription(ctx)
			if e != nil {
				return nil, e
			}
			return BillingStatusToResponse(&v), nil
		})
		addMCPTool(server, "billing_checkout", "Create hosted Stripe Checkout and return its URL", false, billing, false, func(ctx context.Context, in mcpQuote) (any, error) {
			url, e := billing.commander.Checkout(ctx, in.Prompts)
			return map[string]string{"url": url}, e
		})
		addMCPTool(server, "billing_confirm_checkout", "Reconcile a completed checkout session", false, billing, false, func(ctx context.Context, in mcpCheckoutConfirm) (any, error) {
			v, e := billing.commander.ConfirmCheckout(ctx, in.SessionID)
			if e != nil {
				return nil, e
			}
			return BillingStatusToResponse(&v), nil
		})
		addMCPTool(server, "billing_change_allowance", "Change prompt allowance with idempotency key", false, billing, false, func(ctx context.Context, in mcpChange) (any, error) {
			if !validIdempotencyKey(in.IdempotencyKey) {
				return nil, koidomain.Validation(koidomain.FieldError{Field: "idempotencyKey", Code: koidomain.FieldInvalid})
			}
			v, e := billing.commander.ChangeAllowance(ctx, in.Prompts, in.IdempotencyKey)
			if e != nil {
				return nil, e
			}
			return BillingChangeToResponse(&v), nil
		})
		addMCPTool(server, "billing_portal", "Get hosted billing portal URL", false, billing, false, func(ctx context.Context, _ struct{}) (any, error) {
			url, e := billing.commander.Portal(ctx)
			return map[string]string{"url": url}, e
		})
	}
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
}
