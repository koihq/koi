package koistripe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/stripe/stripe-go/v87"

	"github.com/koihq/koi/pkg/koipricing"
)

type Client struct {
	stripe *stripe.Client
}

func New(secret string) *Client {
	return newClient(secret, nil)
}

func newClient(secret string, backend *stripe.BackendConfig) *Client {
	if backend == nil {
		backend = &stripe.BackendConfig{HTTPClient: &http.Client{Timeout: 15 * time.Second}}
	}
	return &Client{
		stripe: stripe.NewClient(secret, stripe.WithBackends(stripe.NewBackendsWithConfig(backend))),
	}
}

type Price struct {
	ID            string
	Active        bool
	Currency      string
	BillingScheme string
	TaxBehavior   string
	TiersMode     string
	Tiers         []Tier
	Recurring     struct {
		Interval      string
		IntervalCount int64
		UsageType     string
	}
}

type Tier struct {
	UpTo       *int
	FlatAmount int64
	UnitAmount *int64
}

func (p Price) ValidateCatalog(catalog koipricing.Catalog) error {
	if !p.Active || p.Currency != catalog.Currency || p.BillingScheme != "tiered" ||
		p.TiersMode != "volume" || p.TaxBehavior != "inclusive" ||
		p.Recurring.Interval != "month" || p.Recurring.IntervalCount != 1 ||
		p.Recurring.UsageType != "licensed" || len(p.Tiers) != len(catalog.Options) {
		return fmt.Errorf("Stripe price %s does not match the monthly catalog", p.ID)
	}
	for index, option := range catalog.Options {
		tier := p.Tiers[index]
		if tier.FlatAmount != int64(option.Euros)*100 || tier.UnitAmount != nil ||
			(index == len(catalog.Options)-1 && tier.UpTo != nil) ||
			(index < len(catalog.Options)-1 && (tier.UpTo == nil || *tier.UpTo != option.Prompts)) {
			return fmt.Errorf("Stripe price %s has an invalid tier for %d prompts", p.ID, option.Prompts)
		}
	}
	return nil
}

func (p Price) MonthlyAmount(prompts int) (int64, error) {
	if _, err := koipricing.Current().Quote(prompts); err != nil {
		return 0, err
	}
	for _, tier := range p.Tiers {
		if tier.UpTo == nil || prompts <= *tier.UpTo {
			return tier.FlatAmount, nil
		}
	}
	return 0, errors.New("Stripe price has no matching tier")
}

func (c *Client) GetPrice(ctx context.Context, id string) (Price, error) {
	params := &stripe.PriceRetrieveParams{}
	params.AddExpand("tiers")
	got, err := c.stripe.V1Prices.Retrieve(ctx, id, params)
	if err != nil {
		return Price{}, err
	}
	if got == nil || got.ID != id {
		return Price{}, errors.New("Stripe returned unexpected price")
	}

	price := Price{
		ID:            got.ID,
		Active:        got.Active,
		Currency:      string(got.Currency),
		BillingScheme: string(got.BillingScheme),
		TaxBehavior:   string(got.TaxBehavior),
		TiersMode:     string(got.TiersMode),
	}
	if got.Recurring != nil {
		price.Recurring.Interval = string(got.Recurring.Interval)
		price.Recurring.IntervalCount = got.Recurring.IntervalCount
		price.Recurring.UsageType = string(got.Recurring.UsageType)
	}
	if got.LastResponse == nil {
		return Price{}, errors.New("Stripe price response is missing")
	}
	var raw struct {
		Tiers []struct {
			UnitAmount *int64 `json:"unit_amount"`
		} `json:"tiers"`
	}
	if err := json.Unmarshal(got.LastResponse.RawJSON, &raw); err != nil || len(raw.Tiers) != len(got.Tiers) {
		return Price{}, errors.New("Stripe price tiers are incomplete")
	}
	for index, tier := range got.Tiers {
		if tier == nil {
			return Price{}, errors.New("Stripe returned an empty price tier")
		}
		item := Tier{FlatAmount: tier.FlatAmount, UnitAmount: raw.Tiers[index].UnitAmount}
		if tier.UpTo > 0 {
			upTo := int(tier.UpTo)
			item.UpTo = &upTo
		}
		price.Tiers = append(price.Tiers, item)
	}
	return price, nil
}

func (c *Client) CreateCustomer(ctx context.Context, organizationID string) (string, error) {
	params := &stripe.CustomerCreateParams{
		Metadata: map[string]string{"organization_id": organizationID},
	}
	params.SetIdempotencyKey("koi-customer-" + organizationID)
	got, err := c.stripe.V1Customers.Create(ctx, params)
	if err != nil {
		return "", err
	}
	if got == nil || got.ID == "" {
		return "", errors.New("Stripe returned an empty customer")
	}

	return got.ID, nil
}

type CheckoutSession struct {
	ID           string `json:"id"`
	URL          string `json:"url"`
	Customer     string `json:"customer"`
	Subscription string `json:"subscription"`
	Reference    string `json:"client_reference_id"`
	ExpiresAt    int64  `json:"expires_at"`
}

func (c *Client) CreateCheckout(ctx context.Context, customerID, organizationID, priceID, reservationID string, quantity int, publicURL string, trial bool) (CheckoutSession, error) {
	params := &stripe.CheckoutSessionCreateParams{
		Mode:                    stripe.String("subscription"),
		UIMode:                  stripe.String("hosted_page"),
		Customer:                stripe.String(customerID),
		PaymentMethodCollection: stripe.String("always"),
		ClientReferenceID:       stripe.String(organizationID),
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{
			{Price: stripe.String(priceID), Quantity: stripe.Int64(int64(quantity))},
		},
		SubscriptionData: &stripe.CheckoutSessionCreateSubscriptionDataParams{
			Metadata: map[string]string{"organization_id": organizationID},
		},
		SuccessURL: stripe.String(publicURL + "/settings/billing?checkout=success"),
		CancelURL:  stripe.String(publicURL + "/settings/billing?checkout=cancel"),
	}
	if trial {
		params.SubscriptionData.TrialPeriodDays = stripe.Int64(7)
	}
	params.SetIdempotencyKey("koi-checkout-" + reservationID)

	got, err := c.stripe.V1CheckoutSessions.Create(ctx, params)
	if err != nil {
		return CheckoutSession{}, err
	}
	if got == nil || got.Customer == nil || got.ID == "" || got.URL == "" ||
		got.Customer.ID != customerID || got.ClientReferenceID != organizationID ||
		got.ExpiresAt <= time.Now().Unix() {
		return CheckoutSession{}, errors.New("Stripe returned an invalid Checkout Session")
	}

	return CheckoutSession{
		ID:        got.ID,
		URL:       got.URL,
		Customer:  got.Customer.ID,
		Reference: got.ClientReferenceID,
		ExpiresAt: got.ExpiresAt,
	}, nil
}

func (c *Client) CreatePortal(ctx context.Context, customerID, publicURL string) (string, error) {
	got, err := c.stripe.V1BillingPortalSessions.Create(ctx, &stripe.BillingPortalSessionCreateParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(publicURL + "/settings/billing"),
	})
	if err != nil {
		return "", err
	}
	if got == nil || got.URL == "" {
		return "", errors.New("Stripe returned an empty portal URL")
	}

	return got.URL, nil
}

type Subscription struct {
	ID                string            `json:"id"`
	Customer          string            `json:"customer"`
	Status            string            `json:"status"`
	Metadata          map[string]string `json:"metadata"`
	CurrentPeriodEnd  int64             `json:"current_period_end"`
	TrialEnd          *int64            `json:"trial_end"`
	CancelAtPeriodEnd bool              `json:"cancel_at_period_end"`
	Schedule          *string           `json:"schedule"`
	Items             struct {
		Data []struct {
			ID               string `json:"id"`
			Quantity         int    `json:"quantity"`
			CurrentPeriodEnd int64  `json:"current_period_end"`
			Price            struct {
				ID string `json:"id"`
			} `json:"price"`
		} `json:"data"`
	} `json:"items"`
}

func subscriptionFromStripe(got *stripe.Subscription) (Subscription, error) {
	if got == nil || got.Customer == nil || got.Items == nil {
		return Subscription{}, errors.New("Stripe returned an incomplete subscription")
	}
	result := Subscription{
		ID:                got.ID,
		Customer:          got.Customer.ID,
		Status:            string(got.Status),
		Metadata:          got.Metadata,
		CancelAtPeriodEnd: got.CancelAtPeriodEnd,
	}
	if got.TrialEnd > 0 {
		result.TrialEnd = &got.TrialEnd
	}
	if got.Schedule != nil {
		result.Schedule = &got.Schedule.ID
	}
	for _, item := range got.Items.Data {
		if item == nil || item.Price == nil {
			return Subscription{}, errors.New("Stripe returned an incomplete subscription item")
		}
		result.Items.Data = append(result.Items.Data, struct {
			ID               string `json:"id"`
			Quantity         int    `json:"quantity"`
			CurrentPeriodEnd int64  `json:"current_period_end"`
			Price            struct {
				ID string `json:"id"`
			} `json:"price"`
		}{
			ID:               item.ID,
			Quantity:         int(item.Quantity),
			CurrentPeriodEnd: item.CurrentPeriodEnd,
			Price: struct {
				ID string `json:"id"`
			}{ID: item.Price.ID},
		})
		if item.CurrentPeriodEnd > result.CurrentPeriodEnd {
			result.CurrentPeriodEnd = item.CurrentPeriodEnd
		}
	}

	return result, nil
}

func (c *Client) GetSubscription(ctx context.Context, id string) (Subscription, error) {
	got, err := c.stripe.V1Subscriptions.Retrieve(ctx, id, nil)
	if err != nil {
		return Subscription{}, err
	}
	result, err := subscriptionFromStripe(got)
	if err != nil {
		return Subscription{}, err
	}
	if result.ID != id {
		return Subscription{}, errors.New("Stripe returned unexpected subscription")
	}

	return result, nil
}

func (c *Client) Upgrade(ctx context.Context, subscriptionID, itemID string, quantity int) (Subscription, error) {
	params := &stripe.SubscriptionUpdateParams{
		Items: []*stripe.SubscriptionUpdateItemParams{
			{ID: stripe.String(itemID), Quantity: stripe.Int64(int64(quantity))},
		},
		PaymentBehavior:   stripe.String("pending_if_incomplete"),
		ProrationBehavior: stripe.String("always_invoice"),
	}
	got, err := c.stripe.V1Subscriptions.Update(ctx, subscriptionID, params)
	if err != nil {
		return Subscription{}, err
	}
	result, err := subscriptionFromStripe(got)
	if err != nil {
		return Subscription{}, err
	}
	if result.ID != subscriptionID {
		return Subscription{}, errors.New("Stripe returned unexpected subscription")
	}

	return result, nil
}

type Schedule struct {
	ID           string
	CurrentPhase *struct {
		StartDate int64
		EndDate   int64
	}
	Phases []struct {
		StartDate int64
		EndDate   int64
	}
}

func scheduleFromStripe(got *stripe.SubscriptionSchedule) (Schedule, error) {
	if got == nil || got.ID == "" {
		return Schedule{}, errors.New("Stripe returned an empty schedule")
	}
	result := Schedule{ID: got.ID}
	if got.CurrentPhase != nil {
		result.CurrentPhase = &struct {
			StartDate int64
			EndDate   int64
		}{
			StartDate: got.CurrentPhase.StartDate,
			EndDate:   got.CurrentPhase.EndDate,
		}
	}
	for _, phase := range got.Phases {
		if phase != nil {
			result.Phases = append(result.Phases, struct {
				StartDate int64
				EndDate   int64
			}{StartDate: phase.StartDate, EndDate: phase.EndDate})
		}
	}

	return result, nil
}

func (c *Client) CreateSchedule(ctx context.Context, subscriptionID string) (Schedule, error) {
	got, err := c.stripe.V1SubscriptionSchedules.Create(ctx, &stripe.SubscriptionScheduleCreateParams{
		FromSubscription: stripe.String(subscriptionID),
	})
	if err != nil {
		return Schedule{}, err
	}

	return scheduleFromStripe(got)
}

func (c *Client) GetSchedule(ctx context.Context, id string) (Schedule, error) {
	got, err := c.stripe.V1SubscriptionSchedules.Retrieve(ctx, id, nil)
	if err != nil {
		return Schedule{}, err
	}
	result, err := scheduleFromStripe(got)
	if err != nil {
		return Schedule{}, err
	}
	if result.ID != id {
		return Schedule{}, errors.New("Stripe returned unexpected schedule")
	}

	return result, nil
}

func (c *Client) ScheduleDowngrade(ctx context.Context, schedule Schedule, priceID string, currentQuantity, nextQuantity int, periodEnd int64) error {
	startDate := int64(0)
	if schedule.CurrentPhase != nil {
		startDate = schedule.CurrentPhase.StartDate
	} else {
		for _, phase := range schedule.Phases {
			if phase.EndDate == periodEnd {
				startDate = phase.StartDate
				break
			}
		}
	}
	if startDate == 0 || periodEnd == 0 || startDate >= periodEnd {
		return errors.New("Stripe schedule has no current phase")
	}

	params := &stripe.SubscriptionScheduleUpdateParams{
		EndBehavior:       stripe.String("release"),
		ProrationBehavior: stripe.String("none"),
		Phases: []*stripe.SubscriptionScheduleUpdatePhaseParams{
			{
				StartDate: stripe.Int64(startDate),
				EndDate:   stripe.Int64(periodEnd),
				Items: []*stripe.SubscriptionScheduleUpdatePhaseItemParams{
					{Price: stripe.String(priceID), Quantity: stripe.Int64(int64(currentQuantity))},
				},
			},
			{
				StartDate: stripe.Int64(periodEnd),
				Duration: &stripe.SubscriptionScheduleUpdatePhaseDurationParams{
					Interval:      stripe.String("month"),
					IntervalCount: stripe.Int64(1),
				},
				Items: []*stripe.SubscriptionScheduleUpdatePhaseItemParams{
					{Price: stripe.String(priceID), Quantity: stripe.Int64(int64(nextQuantity))},
				},
			},
		},
	}
	_, err := c.stripe.V1SubscriptionSchedules.Update(ctx, schedule.ID, params)
	return err
}
