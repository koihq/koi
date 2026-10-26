package koistripe

import (
	"encoding/json"
	"errors"

	"github.com/stripe/stripe-go/v87"

	"github.com/koihq/koi/pkg/koidomain"
)

var ErrInvalidSignature = errors.New("invalid Stripe webhook signature")

func VerifyEvent(payload []byte, header, secret string) (koidomain.BillingEvent, error) {
	if err := stripe.ValidatePayload(payload, header, secret); err != nil {
		return koidomain.BillingEvent{}, ErrInvalidSignature
	}

	var event stripe.Event
	if err := json.Unmarshal(payload, &event); err != nil || event.ID == "" || event.Type == "" || event.Data == nil {
		return koidomain.BillingEvent{}, errors.New("invalid Stripe event")
	}

	result := koidomain.BillingEvent{ID: event.ID, Type: string(event.Type)}
	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			return result, err
		}
		if session.ID == "" || session.Customer == nil || session.Subscription == nil {
			return result, errors.New("incomplete Stripe checkout event")
		}
		result.CheckoutID = session.ID
		result.CustomerID = session.Customer.ID
		result.OrganizationID = session.ClientReferenceID
		result.SubscriptionID = session.Subscription.ID
	case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted",
		"customer.subscription.pending_update_applied", "customer.subscription.pending_update_expired":
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			return result, err
		}
		if subscription.ID == "" || subscription.Customer == nil {
			return result, errors.New("incomplete Stripe subscription event")
		}
		result.CustomerID = subscription.Customer.ID
		result.OrganizationID = subscription.Metadata["organization_id"]
		result.SubscriptionID = subscription.ID
	}

	return result, nil
}
