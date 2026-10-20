package koistripe

import (
	"encoding/json"
	"errors"

	"github.com/stripe/stripe-go/v87"
)

var ErrInvalidSignature = errors.New("invalid Stripe webhook signature")

type Event struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
}

func VerifyEvent(payload []byte, header, secret string) (Event, error) {
	if err := stripe.ValidatePayload(payload, header, secret); err != nil {
		return Event{}, ErrInvalidSignature
	}

	var event Event
	if err := json.Unmarshal(payload, &event); err != nil || event.ID == "" || event.Type == "" {
		return Event{}, errors.New("invalid Stripe event")
	}

	return event, nil
}
