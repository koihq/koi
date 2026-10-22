package koiapi

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/koistripe"
)

type recordingBillingSync struct {
	BillingRepository
	priceID string
	prompts int
}

func (r *recordingBillingSync) SyncSubscription(_ context.Context, _, _, _, _, priceID, _ string, prompts int, _, _ *time.Time, _ bool) error {
	r.priceID = priceID
	r.prompts = prompts
	return nil
}

func TestSubscriptionKeepsStripeQuantityAsPromptAllowance(t *testing.T) {
	var subscription koistripe.Subscription
	require.NoError(t, json.Unmarshal([]byte(`{"id":"sub_1","customer":"cus_1","metadata":{"organization_id":"org_1"},"items":{"data":[{"quantity":10,"price":{"id":"price_test"}}]}}`), &subscription))
	repo := &recordingBillingSync{}
	handler := &BillingHandler{repo: repo, priceID: "price_test"}
	require.NoError(t, handler.syncSubscription(t.Context(), "evt_1", subscription))
	require.Equal(t, "price_test", repo.priceID)
	require.Equal(t, 10, repo.prompts)

	subscription.Items.Data[0].Price.ID = "price_other"
	require.Error(t, handler.syncSubscription(t.Context(), "evt_2", subscription))
}
