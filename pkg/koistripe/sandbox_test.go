package koistripe

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v87"
)

// This opt-in suite creates isolated sandbox customers and deletes their test clocks.
// Run: KOI_STRIPE_SANDBOX_TEST=1 go test ./pkg/koistripe -run TestStripeSandbox -v
func TestStripeSandbox(t *testing.T) {
	if os.Getenv("KOI_STRIPE_SANDBOX_TEST") != "1" {
		t.Skip("set KOI_STRIPE_SANDBOX_TEST=1 to run against the configured Stripe sandbox")
	}
	config, err := godotenv.Read("../../.env")
	require.NoError(t, err)
	secret, priceID := config["KOI_STRIPE_SECRET_KEY"], config["KOI_STRIPE_PRICE_ID"]
	require.True(t, strings.HasPrefix(secret, "sk_test_"), "sandbox tests require a test key")
	require.NotEmpty(t, priceID)
	client := New(secret)
	require.NoError(t, client.ValidatePrice(t.Context(), priceID))

	fixture := func(t *testing.T, trial bool) (string, string) {
		t.Helper()
		clock, err := client.stripe.V1TestHelpersTestClocks.Create(t.Context(), &stripe.TestHelpersTestClockCreateParams{
			FrozenTime: stripe.Int64(time.Now().Unix()),
			Name:       stripe.String("Koi billing acceptance " + uuid.NewString()),
		})
		require.NoError(t, err)
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_, err := client.stripe.V1TestHelpersTestClocks.Delete(ctx, clock.ID, nil)
			require.NoError(t, err)
		})
		customer, err := client.stripe.V1Customers.Create(t.Context(), &stripe.CustomerCreateParams{
			Name:      stripe.String("Koi billing acceptance"),
			TestClock: stripe.String(clock.ID),
		})
		require.NoError(t, err)
		method, err := client.stripe.V1PaymentMethods.Attach(t.Context(), "pm_card_visa", &stripe.PaymentMethodAttachParams{Customer: stripe.String(customer.ID)})
		require.NoError(t, err)
		params := &stripe.SubscriptionCreateParams{
			Customer:             stripe.String(customer.ID),
			DefaultPaymentMethod: stripe.String(method.ID),
			Metadata:             map[string]string{"organization_id": uuid.NewString()},
			Items: []*stripe.SubscriptionCreateItemParams{
				{Price: stripe.String(priceID), Quantity: stripe.Int64(25)},
			},
		}
		if trial {
			params.TrialPeriodDays = stripe.Int64(7)
		}
		subscription, err := client.stripe.V1Subscriptions.Create(t.Context(), params)
		require.NoError(t, err)

		return subscription.ID, clock.ID
	}

	t.Run("trial and scheduled downgrade", func(t *testing.T) {
		id, clockID := fixture(t, true)
		current, err := client.GetSubscription(t.Context(), id)
		require.NoError(t, err)
		trialEnd := current.TrialEnd
		changed, err := client.ChangeTrial(t.Context(), current, 10, "koi-test-trial-"+uuid.NewString())
		require.NoError(t, err)
		require.Equal(t, "trialing", changed.Status)
		require.Equal(t, trialEnd, changed.TrialEnd)
		require.Equal(t, 10, changed.Quantity)

		advanceSandboxClock(t, client, clockID, trialEnd.Add(time.Hour).Unix())
		current, err = client.GetSubscription(t.Context(), id)
		require.NoError(t, err)
		require.Equal(t, "active", current.Status)
		scheduled, err := client.ScheduleDowngrade(t.Context(), current, 5, "koi-test-downgrade-"+uuid.NewString())
		require.NoError(t, err)
		require.NotNil(t, scheduled.PendingQuantity)
		require.Equal(t, 5, *scheduled.PendingQuantity)
		require.Equal(t, 10, scheduled.Quantity)

		released, err := client.ReleaseSchedule(t.Context(), scheduled, "koi-test-release-"+uuid.NewString())
		require.NoError(t, err)
		require.Nil(t, released.ScheduleID)
		scheduled, err = client.ScheduleDowngrade(t.Context(), released, 5, "koi-test-renewal-"+uuid.NewString())
		require.NoError(t, err)
		advanceSandboxClock(t, client, clockID, scheduled.CurrentPeriodEnd.Add(time.Hour).Unix())
		current, err = client.GetSubscription(t.Context(), id)
		require.NoError(t, err)
		require.Equal(t, 5, current.Quantity)
		require.Nil(t, current.PendingQuantity)
	})

	for _, card := range []string{"pm_card_visa", "pm_card_chargeCustomerFail", "pm_card_authenticationRequired"} {
		t.Run(card, func(t *testing.T) {
			id, _ := fixture(t, false)
			current, err := client.GetSubscription(t.Context(), id)
			require.NoError(t, err)
			method, err := client.stripe.V1PaymentMethods.Attach(t.Context(), card, &stripe.PaymentMethodAttachParams{Customer: stripe.String(current.CustomerID)})
			require.NoError(t, err)
			_, err = client.stripe.V1Subscriptions.Update(t.Context(), id, &stripe.SubscriptionUpdateParams{DefaultPaymentMethod: stripe.String(method.ID)})
			require.NoError(t, err)
			changed, err := client.Upgrade(t.Context(), current, 50, "koi-test-upgrade-"+uuid.NewString())
			require.NoError(t, err)
			if card == "pm_card_visa" {
				require.Equal(t, 50, changed.Quantity)
				require.Nil(t, changed.PendingUpgradeQuantity)
			} else {
				require.Equal(t, 25, changed.Quantity)
				require.NotNil(t, changed.PendingUpgradeQuantity)
				require.Equal(t, 50, *changed.PendingUpgradeQuantity)
				require.NotEmpty(t, changed.PaymentURL)
				require.NotNil(t, changed.PendingUpgradeExpiresAt)
			}
		})
	}

	t.Run("cancellation retains access until the paid period ends", func(t *testing.T) {
		id, clockID := fixture(t, false)
		_, err := client.stripe.V1Subscriptions.Update(t.Context(), id, &stripe.SubscriptionUpdateParams{CancelAtPeriodEnd: stripe.Bool(true)})
		require.NoError(t, err)
		current, err := client.GetSubscription(t.Context(), id)
		require.NoError(t, err)
		require.True(t, current.CancelAtPeriodEnd)
		require.Equal(t, "active", current.Status)
		advanceSandboxClock(t, client, clockID, current.CurrentPeriodEnd.Add(time.Hour).Unix())
		current, err = client.GetSubscription(t.Context(), id)
		require.NoError(t, err)
		require.Equal(t, "canceled", current.Status)
	})
}

func advanceSandboxClock(t *testing.T, client *Client, id string, frozenTime int64) {
	t.Helper()
	_, err := client.stripe.V1TestHelpersTestClocks.Advance(t.Context(), id, &stripe.TestHelpersTestClockAdvanceParams{FrozenTime: stripe.Int64(frozenTime)})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		clock, err := client.stripe.V1TestHelpersTestClocks.Retrieve(t.Context(), id, nil)
		return err == nil && clock.Status == stripe.TestHelpersTestClockStatusReady
	}, 45*time.Second, time.Second)
}
