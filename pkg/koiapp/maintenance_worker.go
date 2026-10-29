package koiapp

import (
	"context"
	"time"

	"github.com/koihq/koi/pkg/koidatabase"
	"github.com/koihq/koi/pkg/koidomain"
	"github.com/riverqueue/river"
)

type maintenanceArgs struct{}

func (maintenanceArgs) Kind() string { return "koi_maintenance" }

type maintenanceWorker struct {
	river.WorkerDefaults[maintenanceArgs]
	store   *koidatabase.Store
	billing koidomain.BillingCommander
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
	if w.billing != nil {
		return w.billing.ReconcilePendingCheckouts(ctx, time.Now().UTC())
	}
	return nil
}
