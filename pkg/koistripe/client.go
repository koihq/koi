package koistripe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/stripe/stripe-go/v87"

	"github.com/koihq/koi/pkg/koidomain"
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
		backend = &stripe.BackendConfig{
			HTTPClient: &http.Client{
				Timeout: 15 * time.Second,
			},
		}
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
	UpTo              *int
	FlatAmount        int64
	UnitAmount        *int64
	UnitAmountDecimal *string
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
		if tier.FlatAmount != int64(option.Euros)*100 || tier.UnitAmount != nil || tier.UnitAmountDecimal != nil ||
			(index == len(catalog.Options)-1 && tier.UpTo != nil) ||
			(index < len(catalog.Options)-1 && (tier.UpTo == nil || *tier.UpTo != option.Prompts)) {
			return fmt.Errorf("Stripe price %s has an invalid tier for %d prompts", p.ID, option.Prompts)
		}
	}
	return nil
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
			UnitAmount        *int64  `json:"unit_amount"`
			UnitAmountDecimal *string `json:"unit_amount_decimal"`
		} `json:"tiers"`
	}
	if err := json.Unmarshal(got.LastResponse.RawJSON, &raw); err != nil || len(raw.Tiers) != len(got.Tiers) {
		return Price{}, errors.New("Stripe price tiers are incomplete")
	}
	for index, tier := range got.Tiers {
		if tier == nil {
			return Price{}, errors.New("Stripe returned an empty price tier")
		}
		item := Tier{
			FlatAmount:        tier.FlatAmount,
			UnitAmount:        raw.Tiers[index].UnitAmount,
			UnitAmountDecimal: raw.Tiers[index].UnitAmountDecimal,
		}
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

func (c *Client) ValidatePrice(ctx context.Context, id string) error {
	price, err := c.GetPrice(ctx, id)
	if err != nil {
		return err
	}
	return price.ValidateCatalog(koipricing.Current())
}

func (c *Client) CreateCheckout(ctx context.Context, input koidomain.BillingCheckoutInput) (koidomain.BillingCheckout, error) {
	params := &stripe.CheckoutSessionCreateParams{
		Mode:                    stripe.String("subscription"),
		UIMode:                  stripe.String("hosted_page"),
		Customer:                stripe.String(input.CustomerID),
		PaymentMethodCollection: stripe.String("always"),
		ClientReferenceID:       stripe.String(input.OrganizationID),
		ManagedPayments: &stripe.CheckoutSessionCreateManagedPaymentsParams{
			Enabled: stripe.Bool(true),
		},
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{
			{
				Price:    stripe.String(input.PriceID),
				Quantity: stripe.Int64(int64(input.Quantity)),
			},
		},
		SubscriptionData: &stripe.CheckoutSessionCreateSubscriptionDataParams{
			Metadata: map[string]string{
				"organization_id": input.OrganizationID,
			},
		},
		SuccessURL: stripe.String(input.PublicURL + "/settings/billing?checkout=success"),
		CancelURL:  stripe.String(input.PublicURL + "/settings/billing?checkout=cancel"),
	}
	if input.Trial {
		params.SubscriptionData.TrialPeriodDays = stripe.Int64(7)
	}
	params.SetIdempotencyKey("koi-checkout-" + input.ReservationID)

	got, err := c.stripe.V1CheckoutSessions.Create(ctx, params)
	if err != nil {
		return koidomain.BillingCheckout{}, err
	}
	result, err := checkoutFromStripe(got)
	if err != nil {
		return koidomain.BillingCheckout{}, err
	}
	if result.URL == "" || result.CustomerID != input.CustomerID ||
		result.OrganizationID != input.OrganizationID || !result.ExpiresAt.After(time.Now()) {
		return koidomain.BillingCheckout{}, errors.New("Stripe returned an invalid Checkout Session")
	}
	return result, nil
}

func checkoutFromStripe(got *stripe.CheckoutSession) (koidomain.BillingCheckout, error) {
	if got == nil || got.ID == "" || got.Customer == nil {
		return koidomain.BillingCheckout{}, errors.New("Stripe returned an incomplete Checkout Session")
	}
	result := koidomain.BillingCheckout{
		ID:             got.ID,
		URL:            got.URL,
		CustomerID:     got.Customer.ID,
		OrganizationID: got.ClientReferenceID,
		Status:         string(got.Status),
		ExpiresAt:      time.Unix(got.ExpiresAt, 0).UTC(),
	}
	if got.Subscription != nil {
		result.SubscriptionID = got.Subscription.ID
	}
	return result, nil
}

func (c *Client) ExpireCheckout(ctx context.Context, id string) (koidomain.BillingCheckout, error) {
	params := &stripe.CheckoutSessionExpireParams{}
	params.SetIdempotencyKey("koi-expire-" + id)
	got, err := c.stripe.V1CheckoutSessions.Expire(ctx, id, params)
	if err != nil {
		// Expiration races payment completion; retrieve the resulting state before replacing it.
		var stripeError *stripe.Error
		if !errors.As(err, &stripeError) || stripeError.HTTPStatusCode != http.StatusBadRequest {
			return koidomain.BillingCheckout{}, err
		}
		got, err = c.stripe.V1CheckoutSessions.Retrieve(ctx, id, nil)
	}
	if err != nil {
		return koidomain.BillingCheckout{}, err
	}
	result, err := checkoutFromStripe(got)
	if err == nil && result.ID != id {
		return koidomain.BillingCheckout{}, errors.New("Stripe returned unexpected checkout")
	}
	return result, err
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

func timestamp(value int64) *time.Time {
	if value <= 0 {
		return nil
	}
	result := time.Unix(value, 0).UTC()
	return &result
}

func subscriptionFromStripe(got *stripe.Subscription) (koidomain.BillingSnapshot, error) {
	if got == nil || got.ID == "" || got.Customer == nil || got.Items == nil || len(got.Items.Data) != 1 {
		return koidomain.BillingSnapshot{}, errors.New("Stripe subscription must have one price item")
	}
	item := got.Items.Data[0]
	if item == nil || item.Price == nil || item.ID == "" {
		return koidomain.BillingSnapshot{}, errors.New("Stripe returned an incomplete subscription item")
	}
	result := koidomain.BillingSnapshot{
		ID:                got.ID,
		CustomerID:        got.Customer.ID,
		OrganizationID:    got.Metadata["organization_id"],
		ItemID:            item.ID,
		PriceID:           item.Price.ID,
		Quantity:          int(item.Quantity),
		Status:            string(got.Status),
		CancelAtPeriodEnd: got.CancelAtPeriodEnd,
		CurrentPeriodEnd:  timestamp(item.CurrentPeriodEnd),
		TrialEnd:          timestamp(got.TrialEnd),
	}
	if got.Schedule != nil && got.Schedule.ID != "" {
		result.ScheduleID = &got.Schedule.ID
	}
	if got.PendingUpdate != nil {
		for _, pending := range got.PendingUpdate.SubscriptionItems {
			if pending != nil && pending.ID == item.ID {
				quantity := int(pending.Quantity)
				result.PendingUpgradeQuantity = &quantity
				result.PendingUpgradeExpiresAt = timestamp(got.PendingUpdate.ExpiresAt)
			}
		}
	}
	if got.LatestInvoice != nil && (result.PendingUpgradeQuantity != nil ||
		got.Status == stripe.SubscriptionStatusPastDue || got.Status == stripe.SubscriptionStatusUnpaid ||
		got.Status == stripe.SubscriptionStatusIncomplete) {
		result.PaymentURL = got.LatestInvoice.HostedInvoiceURL
	}
	return result, nil
}

func (c *Client) GetSubscription(ctx context.Context, id string) (koidomain.BillingSnapshot, error) {
	params := &stripe.SubscriptionRetrieveParams{}
	params.AddExpand("latest_invoice")
	got, err := c.stripe.V1Subscriptions.Retrieve(ctx, id, params)
	if err != nil {
		return koidomain.BillingSnapshot{}, err
	}
	result, err := subscriptionFromStripe(got)
	if err != nil {
		return koidomain.BillingSnapshot{}, err
	}
	if result.ID != id {
		return koidomain.BillingSnapshot{}, errors.New("Stripe returned unexpected subscription")
	}
	if result.ScheduleID != nil {
		schedule, err := c.stripe.V1SubscriptionSchedules.Retrieve(ctx, *result.ScheduleID, nil)
		if err != nil {
			return koidomain.BillingSnapshot{}, err
		}
		if schedule == nil || schedule.ID != *result.ScheduleID || schedule.Subscription == nil || schedule.Subscription.ID != result.ID {
			return koidomain.BillingSnapshot{}, errors.New("Stripe returned unexpected subscription schedule")
		}
		if schedule.CurrentPhase != nil {
			for _, phase := range schedule.Phases {
				if phase != nil && phase.StartDate >= schedule.CurrentPhase.EndDate &&
					len(phase.Items) == 1 && phase.Items[0] != nil &&
					phase.Items[0].Price != nil && phase.Items[0].Price.ID == result.PriceID {
					quantity := int(phase.Items[0].Quantity)
					if quantity != result.Quantity {
						result.PendingQuantity = &quantity
					}
					break
				}
			}
		}
	}
	return result, nil
}

func (c *Client) updateQuantity(ctx context.Context, current koidomain.BillingSnapshot, quantity int, key string, trial bool) (koidomain.BillingSnapshot, error) {
	params := &stripe.SubscriptionUpdateParams{
		Items: []*stripe.SubscriptionUpdateItemParams{
			{
				ID:       stripe.String(current.ItemID),
				Quantity: stripe.Int64(int64(quantity)),
			},
		},
		PaymentBehavior:   stripe.String("pending_if_incomplete"),
		ProrationBehavior: stripe.String("always_invoice"),
	}
	if trial {
		if current.TrialEnd == nil || !current.TrialEnd.After(time.Now()) {
			return koidomain.BillingSnapshot{}, koidomain.ErrBillingConflict
		}
		params.PaymentBehavior = stripe.String("allow_incomplete")
		params.ProrationBehavior = stripe.String("none")
		params.TrialEnd = stripe.Int64(current.TrialEnd.Unix())
	}
	params.SetIdempotencyKey(key)
	params.AddExpand("latest_invoice")
	got, err := c.stripe.V1Subscriptions.Update(ctx, current.ID, params)
	if err != nil {
		return koidomain.BillingSnapshot{}, err
	}
	result, err := subscriptionFromStripe(got)
	if err == nil && result.ID != current.ID {
		return koidomain.BillingSnapshot{}, errors.New("Stripe returned unexpected subscription")
	}
	return result, err
}

func (c *Client) Upgrade(ctx context.Context, current koidomain.BillingSnapshot, quantity int, key string) (koidomain.BillingSnapshot, error) {
	return c.updateQuantity(ctx, current, quantity, key, false)
}

func (c *Client) ChangeTrial(ctx context.Context, current koidomain.BillingSnapshot, quantity int, key string) (koidomain.BillingSnapshot, error) {
	return c.updateQuantity(ctx, current, quantity, key, true)
}

func (c *Client) ReleaseSchedule(ctx context.Context, current koidomain.BillingSnapshot, key string) (koidomain.BillingSnapshot, error) {
	if current.ScheduleID == nil {
		return current, nil
	}
	params := &stripe.SubscriptionScheduleReleaseParams{}
	params.SetIdempotencyKey(key)
	if _, err := c.stripe.V1SubscriptionSchedules.Release(ctx, *current.ScheduleID, params); err != nil {
		return koidomain.BillingSnapshot{}, err
	}
	return c.GetSubscription(ctx, current.ID)
}

var _ koidomain.BillingGateway = (*Client)(nil)
