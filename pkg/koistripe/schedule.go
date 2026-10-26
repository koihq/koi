package koistripe

import (
	"context"
	"errors"

	"github.com/stripe/stripe-go/v87"

	"github.com/koihq/koi/pkg/koidomain"
)

func (c *Client) ScheduleDowngrade(ctx context.Context, current koidomain.BillingSnapshot, quantity int, key string) (koidomain.BillingSnapshot, error) {
	scheduleID := current.ScheduleID
	if scheduleID == nil {
		params := &stripe.SubscriptionScheduleCreateParams{FromSubscription: stripe.String(current.ID)}
		params.SetIdempotencyKey(key + "-create-schedule")
		created, err := c.stripe.V1SubscriptionSchedules.Create(ctx, params)
		if err != nil {
			return koidomain.BillingSnapshot{}, err
		}
		if created == nil || created.ID == "" {
			return koidomain.BillingSnapshot{}, errors.New("Stripe returned an empty schedule")
		}
		scheduleID = &created.ID
	}

	schedule, err := c.stripe.V1SubscriptionSchedules.Retrieve(ctx, *scheduleID, nil)
	if err != nil {
		return koidomain.BillingSnapshot{}, err
	}
	if schedule == nil || schedule.ID != *scheduleID || schedule.Subscription == nil ||
		schedule.Subscription.ID != current.ID || schedule.CurrentPhase == nil ||
		current.CurrentPeriodEnd == nil {
		return koidomain.BillingSnapshot{}, errors.New("Stripe schedule has no current phase")
	}

	var phase *stripe.SubscriptionSchedulePhase
	for _, candidate := range schedule.Phases {
		if candidate != nil && candidate.StartDate == schedule.CurrentPhase.StartDate {
			phase = candidate
			break
		}
	}
	if phase == nil || phase.StartDate >= current.CurrentPeriodEnd.Unix() ||
		len(phase.Items) != 1 || phase.Items[0] == nil || phase.Items[0].Price == nil ||
		phase.Items[0].Price.ID != current.PriceID || int(phase.Items[0].Quantity) != current.Quantity {
		return koidomain.BillingSnapshot{}, errors.New("Stripe schedule differs from subscription")
	}

	first, err := phaseParams(phase)
	if err != nil {
		return koidomain.BillingSnapshot{}, err
	}
	first.StartDate = stripe.Int64(phase.StartDate)
	first.EndDate = stripe.Int64(current.CurrentPeriodEnd.Unix())

	next, err := phaseParams(phase)
	if err != nil {
		return koidomain.BillingSnapshot{}, err
	}
	next.TrialEnd = nil
	next.Trial = nil
	next.StartDate = first.EndDate
	next.Duration = &stripe.SubscriptionScheduleUpdatePhaseDurationParams{
		Interval:      stripe.String("month"),
		IntervalCount: stripe.Int64(1),
	}
	next.Items[0].Quantity = stripe.Int64(int64(quantity))
	next.ProrationBehavior = stripe.String("none")

	params := &stripe.SubscriptionScheduleUpdateParams{
		EndBehavior:       stripe.String("release"),
		ProrationBehavior: stripe.String("none"),
		Phases:            []*stripe.SubscriptionScheduleUpdatePhaseParams{first, next},
	}
	params.SetIdempotencyKey(key + "-update-schedule")
	if _, err := c.stripe.V1SubscriptionSchedules.Update(ctx, schedule.ID, params); err != nil {
		return koidomain.BillingSnapshot{}, err
	}

	return c.GetSubscription(ctx, current.ID)
}

// Keep the existing phase's billing configuration when replacing its future quantity.
// Koi does not create Connect fees, thresholds or additional invoice items; reject those
// configurations rather than silently clearing them during an allowance change.
func phaseParams(phase *stripe.SubscriptionSchedulePhase) (*stripe.SubscriptionScheduleUpdatePhaseParams, error) {
	if len(phase.AddInvoiceItems) > 0 || phase.ApplicationFeePercent != 0 ||
		phase.BillingThresholds != nil || phase.OnBehalfOf != nil || phase.TransferData != nil {
		return nil, koidomain.ErrBillingConflict
	}

	params := &stripe.SubscriptionScheduleUpdatePhaseParams{
		Metadata: phase.Metadata,
		Items:    make([]*stripe.SubscriptionScheduleUpdatePhaseItemParams, 0, len(phase.Items)),
	}
	if phase.Currency != "" {
		params.Currency = stripe.String(string(phase.Currency))
	}
	if phase.Description != "" {
		params.Description = stripe.String(phase.Description)
	}
	if phase.BillingCycleAnchor != "" {
		params.BillingCycleAnchor = stripe.String(string(phase.BillingCycleAnchor))
	}
	if phase.ProrationBehavior != "" {
		params.ProrationBehavior = stripe.String(string(phase.ProrationBehavior))
	}
	if phase.CollectionMethod != nil {
		params.CollectionMethod = stripe.String(string(*phase.CollectionMethod))
	}
	if phase.DefaultPaymentMethod != nil {
		params.DefaultPaymentMethod = stripe.String(phase.DefaultPaymentMethod.ID)
	}
	for _, rate := range phase.DefaultTaxRates {
		if rate != nil {
			params.DefaultTaxRates = append(params.DefaultTaxRates, stripe.String(rate.ID))
		}
	}
	if phase.TrialEnd > 0 {
		params.TrialEnd = stripe.Int64(phase.TrialEnd)
	}
	if phase.Trial {
		params.Trial = stripe.Bool(true)
	}
	if tax := phase.AutomaticTax; tax != nil {
		params.AutomaticTax = &stripe.SubscriptionScheduleUpdatePhaseAutomaticTaxParams{Enabled: stripe.Bool(tax.Enabled)}
		if tax.Liability != nil {
			params.AutomaticTax.Liability = &stripe.SubscriptionScheduleUpdatePhaseAutomaticTaxLiabilityParams{
				Type: stripe.String(string(tax.Liability.Type)),
			}
			if tax.Liability.Account != nil {
				params.AutomaticTax.Liability.Account = stripe.String(tax.Liability.Account.ID)
			}
		}
	}
	for _, discount := range phase.Discounts {
		if discount == nil {
			continue
		}
		item := &stripe.SubscriptionScheduleUpdatePhaseDiscountParams{}
		switch {
		case discount.Discount != nil:
			item.Discount = stripe.String(discount.Discount.ID)
		case discount.PromotionCode != nil:
			item.PromotionCode = stripe.String(discount.PromotionCode.ID)
		case discount.Coupon != nil:
			item.Coupon = stripe.String(discount.Coupon.ID)
		}
		params.Discounts = append(params.Discounts, item)
	}
	if invoice := phase.InvoiceSettings; invoice != nil {
		settings := &stripe.SubscriptionScheduleUpdatePhaseInvoiceSettingsParams{}
		for _, taxID := range invoice.AccountTaxIDs {
			if taxID != nil {
				settings.AccountTaxIDs = append(settings.AccountTaxIDs, stripe.String(taxID.ID))
			}
		}
		for _, field := range invoice.CustomFields {
			if field != nil {
				settings.CustomFields = append(settings.CustomFields, &stripe.SubscriptionScheduleUpdatePhaseInvoiceSettingsCustomFieldParams{
					Name:  stripe.String(field.Name),
					Value: stripe.String(field.Value),
				})
			}
		}
		if invoice.DaysUntilDue > 0 {
			settings.DaysUntilDue = stripe.Int64(invoice.DaysUntilDue)
		}
		if invoice.Description != "" {
			settings.Description = stripe.String(invoice.Description)
		}
		if invoice.Footer != "" {
			settings.Footer = stripe.String(invoice.Footer)
		}
		if invoice.Issuer != nil {
			settings.Issuer = &stripe.SubscriptionScheduleUpdatePhaseInvoiceSettingsIssuerParams{Type: stripe.String(string(invoice.Issuer.Type))}
			if invoice.Issuer.Account != nil {
				settings.Issuer.Account = stripe.String(invoice.Issuer.Account.ID)
			}
		}
		params.InvoiceSettings = settings
	}
	for _, item := range phase.Items {
		if item == nil || item.Price == nil || item.BillingThresholds != nil || len(item.Discounts) > 0 {
			return nil, koidomain.ErrBillingConflict
		}
		mapped := &stripe.SubscriptionScheduleUpdatePhaseItemParams{
			Price:    stripe.String(item.Price.ID),
			Quantity: stripe.Int64(item.Quantity),
			Metadata: item.Metadata,
		}
		for _, rate := range item.TaxRates {
			if rate != nil {
				mapped.TaxRates = append(mapped.TaxRates, stripe.String(rate.ID))
			}
		}
		params.Items = append(params.Items, mapped)
	}

	return params, nil
}
