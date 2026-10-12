package koiapp

import (
	"context"
	"time"

	"github.com/koihq/koi/pkg/koidatabase"
	"github.com/riverqueue/river"
)

type maintenanceArgs struct{}

func (maintenanceArgs) Kind() string { return "koi_maintenance" }

type maintenanceWorker struct {
	river.WorkerDefaults[maintenanceArgs]
	store *koidatabase.Store
}

func (w *maintenanceWorker) Work(ctx context.Context, _ *river.Job[maintenanceArgs]) error {
	if err := w.store.ReconcileGenerationRuns(ctx); err != nil {
		return err
	}

	if err := w.store.ReconcileVisibilityRuns(ctx); err != nil {
		return err
	}

	return w.store.DispatchVisibility(ctx, time.Now().UTC())
}
