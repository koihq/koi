package app

import (
	"context"
	"time"

	"github.com/koihq/koi/pkg/database"
	"github.com/koihq/koi/pkg/domain"
	"github.com/riverqueue/river"
)

type maintenanceArgs struct{}

func (maintenanceArgs) Kind() string { return "koi_maintenance" }

type maintenanceWorker struct {
	river.WorkerDefaults[maintenanceArgs]
	store   *database.Store
	billing domain.BillingCommander
}

func (w *maintenanceWorker) Work(ctx context.Context, _ *river.Job[maintenanceArgs]) error {
	if err := w.store.ReconcileGenerationRuns(ctx); err != nil {
		return err
	}

	if err := w.store.ReconcileVisibilityRuns(ctx); err != nil {
		return err
	}

	if err := w.store.DispatchVisibility(ctx, time.Now().UTC()); err != nil {
		return err
	}
	if err := w.store.CredentialRepo().CleanupExpired(ctx, time.Now().UTC()); err != nil {
		return err
	}
	if w.billing != nil {
		return w.billing.ReconcilePendingCheckouts(ctx, time.Now().UTC())
	}
	return nil
}
