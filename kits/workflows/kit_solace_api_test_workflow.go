package workflows

import (
	"fmt"
	"kits-worker/kits/activities"
	"kits-worker/kits/models"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func KitSolaceAPIWorkflow(
	ctx workflow.Context,
	kitName string,
	kit models.Kit,
	auth models.BasicAuth,
) error {

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second * 5,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    2, // only 2 retries
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	type mateJob struct {
		dc, mate string
		ep       models.Endpoint
	}

	jobs := []mateJob{
		{"dc1", "mate1", kit.DC1.Mate1},
		{"dc1", "mate2", kit.DC1.Mate2},
		{"dc2", "mate1", kit.DC2.Mate1},
		{"dc2", "mate2", kit.DC2.Mate2},
	}

	// Execute all mates in parallel
	futures := make([]workflow.Future, 0, len(jobs))
	for _, job := range jobs {
		j := job // capture loop variable
		f := workflow.ExecuteActivity(
			ctx,
			activities.BrokerSEMPApiAboutActivity,
			kitName,
			j.dc,
			j.mate,
			j.ep,
			auth,
		)
		futures = append(futures, f)
	}

	// Track success count
	successCount := 0

	for i, f := range futures {
		var about *activities.AboutResponse
		job := jobs[i]
		if err := f.Get(ctx, &about); err != nil {
			return fmt.Errorf(
				"BrokerSEMPApiAboutActivity failed for kit=%s dc=%s mate=%s host=%s: %w",
				kitName, job.dc, job.mate, job.ep.Host, err,
			)
		}
		successCount++
	}

	// Final summary log only
	workflow.GetLogger(ctx).Info(
		fmt.Sprintf("Success %d/%d for kit %s", successCount, len(jobs), kitName),
	)

	return nil
}
