package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/koihq/koi/pkg/domain"
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
type mcpPromptList struct {
	mcpPage
	Confirmed *bool `json:"confirmed,omitempty"`
	Inactive  *bool `json:"inactive,omitempty"`
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
	BrandID        mcpUUID `json:"brandId"`
	IdempotencyKey string  `json:"idempotencyKey"`
}
type mcpAnalysis struct {
	BrandID mcpUUID           `json:"brandId"`
	Days    int               `json:"days,omitempty"`
	Engines []domain.EngineID `json:"engines,omitempty"`
}
type mcpPromptHistory struct {
	BrandID  mcpUUID `json:"brandId"`
	PromptID mcpUUID `json:"promptId"`
	Page     int     `json:"page,omitempty"`
	PageSize int     `json:"pageSize,omitempty"`
}
type mcpPromptVisibility struct {
	BrandID  mcpUUID           `json:"brandId"`
	PromptID mcpUUID           `json:"promptId"`
	Days     int               `json:"days,omitempty"`
	Engines  []domain.EngineID `json:"engines,omitempty"`
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

func mcpPageParams(page, size int) (domain.PageParams, error) {
	if page == 0 {
		page = 1
	}
	if size == 0 {
		size = 10
	}
	if page < 1 {
		return domain.PageParams{}, domain.Validation(domain.FieldError{Field: "page", Code: domain.FieldInvalid})
	}
	if size < 1 || size > 100 {
		return domain.PageParams{}, domain.Validation(domain.FieldError{Field: "pageSize", Code: domain.FieldInvalid})
	}
	return domain.PageParams{Page: page, PageSize: size}, nil
}

// args validates the brand and page shared by every brand-scoped list tool.
func (in mcpPage) args() (domain.BrandFilter, domain.PageParams, error) {
	if err := requireBrand(in.BrandID.UUID()); err != nil {
		return domain.BrandFilter{}, domain.PageParams{}, err
	}

	page, err := mcpPageParams(in.Page, in.PageSize)
	if err != nil {
		return domain.BrandFilter{}, domain.PageParams{}, err
	}

	return domain.BrandFilter{BrandID: in.BrandID.UUID()}, page, nil
}

// mcpResult maps a domain call's result to its public form: mcpResult(ToRes)(call(ctx, id)).
func mcpResult[T, R any](mapper func(*T) R) func(*T, error) (any, error) {
	return func(entity *T, err error) (any, error) {
		if err != nil {
			return nil, err
		}

		return mapper(entity), nil
	}
}

func mcpPageResult[T, R any](mapper func(*T) R) func(*domain.Page[T], error) (any, error) {
	return func(page *domain.Page[T], err error) (any, error) {
		if err != nil {
			return nil, err
		}

		return mapPage(page, mapper), nil
	}
}
func mcpAccess(ctx context.Context, billing *BillingHandler) error {
	if billing == nil {
		return nil
	}
	return domain.RequireProductAccess(ctx, billing.querier)
}
func addMCPTool[In any](server *mcp.Server, name, description string, readOnly bool, billing *BillingHandler, guard bool, fn func(context.Context, In) (any, error)) {
	mcp.AddTool(server, &mcp.Tool{Name: name, Description: description, Meta: mcp.Meta{"koi/scope": map[bool]string{true: "koi:read", false: "koi:write"}[readOnly]}, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: readOnly}}, func(ctx context.Context, _ *mcp.CallToolRequest, input In) (*mcp.CallToolResult, any, error) {
		scope, ok := ctx.Value(mcpScopeKey{}).(string)
		if !ok {
			return nil, nil, domain.ErrUnauthorized
		}
		required := "koi:write"
		if readOnly {
			required = "koi:read"
		}
		if !slices.Contains(strings.Fields(scope), required) {
			return nil, nil, domain.ErrUnauthorized
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

func newMCPHandler(shared services, billing *BillingHandler) http.Handler {
	server := mcp.NewServer(
		&mcp.Implementation{Name: "koi", Version: "1.0.0"},
		&mcp.ServerOptions{Instructions: "Koi measures a brand's visibility in AI answers. Use brands_list to find the brand and pass its explicit brandId to brand-scoped tools; the browser's active brand does not scope MCP calls. analysis_summary aggregates a selected period and engines, while runs_results contains individual answers and citations; distinguish observed results from recommendations. Each brand gets one asynchronous draft: after generations_create, check generations_get with the returned ID, then call tracking_start to begin collection. On Koi Cloud, an organization that never had a trial gets one free check on up to 5 prompts: billing_subscription reports it as firstLook, runs_results fills in while it runs, and report those answers to the user as soon as they arrive. Daily collection then needs the trial; create a billing_checkout link only if the user asks, and let the user complete the payment. Reuse the same idempotencyKey when retrying generations_create or billing_change_allowance. Billing tools are available only on Koi Cloud."},
	)
	addMCPTool(server, "account_me", "Get your account and organization", true, billing, false, func(ctx context.Context, _ struct{}) (any, error) {
		return mcpResult(AuthToRes)(shared.auth.Me(ctx, uuid.Nil))
	})
	addMCPTool(server, "brands_countries", "Search supported country codes", true, billing, false, func(ctx context.Context, in struct {
		Query string `json:"query"`
	}) (any, error) {
		return domain.SearchSupportedCountries(in.Query), nil
	})
	addMCPTool(server, "brands_list", "Find brands in your organization by name, domain, or repository; returns a page (default 1, 10 items)", true, billing, false, func(ctx context.Context, in struct {
		Page     int    `json:"page,omitempty"`
		PageSize int    `json:"pageSize,omitempty"`
		Query    string `json:"query,omitempty"`
	}) (any, error) {
		p, e := mcpPageParams(in.Page, in.PageSize)
		if e != nil {
			return nil, e
		}
		return mcpPageResult(BrandToRes)(shared.brands.List(ctx, domain.BrandListFilter{Query: in.Query}, p))
	})
	addMCPTool(server, "brands_get", "Get a brand by UUID", true, billing, false, func(ctx context.Context, in mcpID) (any, error) {
		return mcpResult(BrandToRes)(shared.brands.Get(ctx, in.ID.UUID()))
	})
	addMCPTool(server, "brands_create", "Create a brand", false, billing, false, func(ctx context.Context, in mcpBrandCreate) (any, error) {
		return mcpResult(BrandToRes)(shared.brandCommands.Create(ctx, domain.CreateBrandParams{Name: in.Name, Domain: in.Domain, GitHubRepo: in.GitHubRepo, Country: in.Country, Description: in.Description}))
	})
	addMCPTool(server, "brands_update", "Update a brand", false, billing, true, func(ctx context.Context, in mcpBrandUpdate) (any, error) {
		return mcpResult(BrandToRes)(shared.brandCommands.Update(ctx, in.ID.UUID(), domain.UpdateBrandParams{Name: in.Name, Domain: in.Domain, GitHubRepo: in.GitHubRepo, Country: in.Country, Description: in.Description}))
	})
	addMCPTool(server, "prompts_list", "List prompts for brandId, optionally by confirmed and inactive (stopped); hasResults tells whether a prompt has saved answers; paginated (default page 1, 10 items)", true, billing, true, func(ctx context.Context, in mcpPromptList) (any, error) {
		filter, p, e := in.args()
		if e != nil {
			return nil, e
		}
		return mcpPageResult(PromptToRes)(shared.prompts.List(ctx, domain.PromptFilter{
			BrandFilter: filter,
			Confirmed:   in.Confirmed,
			Inactive:    in.Inactive,
		}, p))
	})
	addMCPTool(server, "prompts_get", "Get a prompt", true, billing, true, func(ctx context.Context, in mcpID) (any, error) {
		return mcpResult(PromptToRes)(shared.prompts.Get(ctx, in.ID.UUID()))
	})
	addMCPTool(server, "prompts_create", "Create a prompt for a brand", false, billing, true, func(ctx context.Context, in mcpPromptCreate) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		return mcpResult(PromptToRes)(shared.promptCommands.Create(ctx, in.BrandID.UUID(), domain.CreatePromptParams{Text: in.Text}))
	})
	addMCPTool(server, "prompts_update", "Update a prompt's text, or set inactive to stop (true) or resume (false) tracking a confirmed prompt; its saved answers stay available", false, billing, true, func(ctx context.Context, in mcpPromptUpdate) (any, error) {
		return mcpResult(PromptToRes)(shared.promptCommands.Update(ctx, in.ID.UUID(), domain.UpdatePromptParams{Text: in.Text, Inactive: in.Inactive}))
	})
	addMCPTool(server, "prompts_delete", "Delete a prompt with no saved answers; a prompt with answers returns resource_conflict, so stop it with prompts_update inactive=true instead", false, billing, true, func(ctx context.Context, in mcpID) (any, error) {
		return map[string]bool{"deleted": true}, shared.promptCommands.Delete(ctx, in.ID.UUID())
	})
	addMCPTool(server, "prompts_confirm", "Confirm a prompt", false, billing, true, func(ctx context.Context, in mcpID) (any, error) {
		return mcpResult(PromptToRes)(shared.promptCommands.Confirm(ctx, in.ID.UUID()))
	})
	addMCPTool(server, "competitors_list", "List current competitors for brandId; paginated (default page 1, 10 items)", true, billing, true, func(ctx context.Context, in mcpPage) (any, error) {
		filter, p, e := in.args()
		if e != nil {
			return nil, e
		}
		return mcpPageResult(CompetitorToRes)(shared.competitors.List(ctx, domain.CompetitorFilter{BrandFilter: filter}, p))
	})
	addMCPTool(server, "competitors_get", "Get a competitor", true, billing, true, func(ctx context.Context, in mcpID) (any, error) {
		return mcpResult(CompetitorToRes)(shared.competitors.Get(ctx, in.ID.UUID()))
	})
	addMCPTool(server, "competitors_search", "Search suggested competitors", true, billing, true, func(ctx context.Context, in mcpSearch) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		return shared.competitors.Search(ctx, in.BrandID.UUID(), in.Query)
	})
	addMCPTool(server, "competitors_create", "Create a competitor", false, billing, true, func(ctx context.Context, in mcpCompetitorCreate) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		return mcpResult(CompetitorToRes)(shared.competitorCommands.Create(ctx, in.BrandID.UUID(), domain.CreateCompetitorParams{Name: in.Name, Domain: in.Domain, GitHubRepo: in.GitHubRepo, Aliases: in.Aliases, AdditionalURLs: in.AdditionalURLs}))
	})
	addMCPTool(server, "competitors_update", "Update a competitor", false, billing, true, func(ctx context.Context, in mcpCompetitorUpdate) (any, error) {
		return mcpResult(CompetitorToRes)(shared.competitorCommands.Update(ctx, in.ID.UUID(), domain.UpdateCompetitorParams{Name: in.Name, Domain: in.Domain, GitHubRepo: in.GitHubRepo, Aliases: in.Aliases, AdditionalURLs: in.AdditionalURLs}))
	})
	addMCPTool(server, "competitors_delete", "Delete a competitor", false, billing, true, func(ctx context.Context, in mcpID) (any, error) {
		return map[string]bool{"deleted": true}, shared.competitorCommands.Delete(ctx, in.ID.UUID())
	})
	addMCPTool(server, "competitors_confirm", "Confirm a competitor", false, billing, true, func(ctx context.Context, in mcpID) (any, error) {
		return mcpResult(CompetitorToRes)(shared.competitorCommands.Confirm(ctx, in.ID.UUID()))
	})
	addMCPTool(server, "generations_create", "Start the brand's one asynchronous draft of prompts and competitors; returns an ID to check with generations_get. After it succeeds, or once tracking starts, add entries with prompts_create and competitors_create. Reuse idempotencyKey on retries", false, billing, true, func(ctx context.Context, in mcpGenerationCreate) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		if !validIdempotencyKey(in.IdempotencyKey) {
			return nil, domain.Validation(domain.FieldError{Field: "idempotencyKey", Code: domain.FieldInvalid})
		}
		v, replay, e := shared.generationCommands.Create(ctx, in.BrandID.UUID(), &in.IdempotencyKey, false)
		if e != nil {
			return nil, e
		}
		return map[string]any{"id": v.ID, "state": v.State, "terminalReplay": replay}, nil
	})
	addMCPTool(server, "generations_get", "Get drafting job status", true, billing, true, func(ctx context.Context, in mcpID) (any, error) {
		return mcpResult(GenerationToRes)(shared.generations.Get(ctx, in.ID.UUID()))
	})
	addMCPTool(server, "generations_list", "List drafting jobs", true, billing, true, func(ctx context.Context, in mcpPage) (any, error) {
		filter, p, e := in.args()
		if e != nil {
			return nil, e
		}
		return mcpPageResult(GenerationToRes)(shared.generations.List(ctx, filter, p))
	})
	addMCPTool(server, "tracking_start", "Confirm the brand's drafted prompts and competitors and start collection: daily, or one free check on Koi Cloud before a trial. Runs once per brand; repeating it changes nothing", false, billing, true, func(ctx context.Context, in mcpScope) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		return map[string]bool{"started": true}, shared.tracking.Start(ctx, in.BrandID.UUID())
	})
	addMCPTool(server, "tracking_get", "Get live collection tracking state for a brand", true, billing, true, func(ctx context.Context, in mcpScope) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		return mcpResult(TrackingStatusToRes)(shared.visibility.Tracking(ctx, in.BrandID.UUID()))
	})
	addMCPTool(server, "runs_list", "List visibility runs", true, billing, true, func(ctx context.Context, in mcpPage) (any, error) {
		filter, p, e := in.args()
		if e != nil {
			return nil, e
		}
		return mcpPageResult(VisibilityRunToRes)(shared.visibility.List(ctx, filter, p))
	})
	addMCPTool(server, "runs_get", "Get visibility run status", true, billing, true, func(ctx context.Context, in mcpID) (any, error) {
		return mcpResult(VisibilityRunToRes)(shared.visibility.Get(ctx, in.ID.UUID()))
	})
	addMCPTool(server, "runs_results", "List individual answers and citations for run ID; paginated (default page 1, 10 items)", true, billing, true, func(ctx context.Context, in struct {
		ID       mcpUUID `json:"id"`
		Page     int     `json:"page,omitempty"`
		PageSize int     `json:"pageSize,omitempty"`
	}) (any, error) {
		p, e := mcpPageParams(in.Page, in.PageSize)
		if e != nil {
			return nil, e
		}
		return mcpPageResult(VisibilityResultToRes)(shared.visibility.ListResults(ctx, in.ID.UUID(), p))
	})
	addMCPTool(server, "analysis_summary", "Aggregate visibility for brandId over days (7, 30, or 90; default 30) and selected engines (default all); empty results are not evidence of visibility", true, billing, true, func(ctx context.Context, in mcpAnalysis) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		days, engines, err := visibilityScopeValues(in.Days, in.Engines)
		if err != nil {
			return nil, err
		}
		return shared.analytics.Summary(ctx, domain.VisibilityScope{
			BrandFilter: domain.BrandFilter{BrandID: in.BrandID.UUID()},
			Days:        days,
			Engines:     engines,
		})
	})
	addMCPTool(server, "prompt_visibility", "Analyze a prompt for brandId over days (7, 30, or 90; default 30) and selected engines (default all)", true, billing, true, func(ctx context.Context, in mcpPromptVisibility) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		days, engines, err := visibilityScopeValues(in.Days, in.Engines)
		if err != nil {
			return nil, err
		}
		return shared.analytics.PromptVisibility(ctx, domain.PromptVisibilityScope{
			VisibilityScope: domain.VisibilityScope{
				BrandFilter: domain.BrandFilter{BrandID: in.BrandID.UUID()},
				Days:        days,
				Engines:     engines,
			},
			PromptID: in.PromptID.UUID(),
		})
	})
	addMCPTool(server, "prompt_history_runs", "List saved runs for a prompt", true, billing, true, func(ctx context.Context, in mcpPromptHistory) (any, error) {
		if e := requireBrand(in.BrandID.UUID()); e != nil {
			return nil, e
		}
		p, e := mcpPageParams(in.Page, in.PageSize)
		if e != nil {
			return nil, e
		}
		return mcpPageResult(PromptHistoryRunToRes)(shared.visibility.ListPromptRuns(ctx, in.BrandID.UUID(), in.PromptID.UUID(), p))
	})
	if billing != nil {
		addMCPTool(server, "billing_catalog", "Get subscription catalog", true, billing, false, func(ctx context.Context, _ struct{}) (any, error) {
			v, e := billing.querier.Catalog(ctx)
			if e != nil {
				return nil, e
			}
			return BillingCatalogToRes(&v), nil
		})
		addMCPTool(server, "billing_quote", "Quote a prompt allowance", true, billing, false, func(ctx context.Context, in mcpQuote) (any, error) {
			v, e := billing.querier.Quote(ctx, in.Prompts)
			if e != nil {
				return nil, e
			}
			return BillingQuoteToRes(&v), nil
		})
		addMCPTool(server, "billing_subscription", "Get reconciled subscription state", true, billing, false, func(ctx context.Context, _ struct{}) (any, error) {
			v, e := billing.querier.Subscription(ctx)
			if e != nil {
				return nil, e
			}
			return BillingStatusToRes(&v), nil
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
			return BillingStatusToRes(&v), nil
		})
		addMCPTool(server, "billing_change_allowance", "Change prompt allowance with idempotency key", false, billing, false, func(ctx context.Context, in mcpChange) (any, error) {
			if !validIdempotencyKey(in.IdempotencyKey) {
				return nil, domain.Validation(domain.FieldError{Field: "idempotencyKey", Code: domain.FieldInvalid})
			}
			v, e := billing.commander.ChangeAllowance(ctx, in.Prompts, in.IdempotencyKey)
			if e != nil {
				return nil, e
			}
			return BillingChangeToRes(&v), nil
		})
		addMCPTool(server, "billing_portal", "Get hosted billing portal URL", false, billing, false, func(ctx context.Context, _ struct{}) (any, error) {
			url, e := billing.commander.Portal(ctx)
			return map[string]string{"url": url}, e
		})
	}
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
}
