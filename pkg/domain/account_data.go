package domain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrAccountSubscriptionLive = errors.New("cancel the Stripe subscription first")
	ErrAccountHasOtherMembers  = errors.New("the organization has other members")
	ErrAccountWorkActive       = errors.New("a draft or collection is still running; try again when it ends")
)

// finishedSubscriptionStatuses are the Stripe states that can no longer charge anyone.
var finishedSubscriptionStatuses = []string{"", "canceled", "incomplete_expired"}

// AccountFacts is what decides whether an account can be deleted now.
type AccountFacts struct {
	OrganizationID     uuid.UUID
	StripeCustomerID   string
	SubscriptionStatus string
	Members            int64
	ActiveWork         int64
}

// AccountDataRepository serves data requests from the instance's administrator. They have no
// request identity, so these queries take the account explicitly.
type AccountDataRepository interface {
	AccountFacts(ctx context.Context, userID uuid.UUID) (AccountFacts, error)
	// ExportAccount returns the account's data by section. It selects columns one by one, so
	// password hashes and token hashes cannot reach the file.
	ExportAccount(ctx context.Context, userID, organizationID uuid.UUID) (map[string]json.RawMessage, error)
	DeleteAccount(ctx context.Context, user *User, organizationID uuid.UUID) error
}

// ExportAccount answers a request for a copy of someone's data.
func ExportAccount(ctx context.Context, store Store, email string) (map[string]json.RawMessage, error) {
	user, err := store.UserRepo().GetByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return nil, err
	}

	facts, err := store.AccountDataRepo().AccountFacts(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return store.AccountDataRepo().ExportAccount(ctx, user.ID, facts.OrganizationID)
}

// DeleteAccount removes an account with its organization, in one transaction. It returns the user,
// so the caller can end their sessions, and the Stripe customer, which is deleted in Stripe by hand.
func DeleteAccount(ctx context.Context, store Store, email string) (uuid.UUID, string, error) {
	var userID uuid.UUID
	var customerID string

	err := store.Atomic(ctx, func(tx Store) error {
		user, err := tx.UserRepo().GetByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
		if err != nil {
			return err
		}

		facts, err := tx.AccountDataRepo().AccountFacts(ctx, user.ID)
		if err != nil {
			return err
		}

		if err := facts.deletable(); err != nil {
			return fmt.Errorf("cannot delete %s: %w", user.Email, err)
		}

		userID = user.ID
		customerID = facts.StripeCustomerID

		return tx.AccountDataRepo().DeleteAccount(ctx, user, facts.OrganizationID)
	})
	if err != nil {
		return uuid.Nil, "", err
	}

	return userID, customerID, nil
}

func (f AccountFacts) deletable() error {
	if !slices.Contains(finishedSubscriptionStatuses, f.SubscriptionStatus) {
		return ErrAccountSubscriptionLive
	}

	if f.Members > 1 {
		return ErrAccountHasOtherMembers
	}

	if f.ActiveWork > 0 {
		return ErrAccountWorkActive
	}

	return nil
}
