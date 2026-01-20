package workflows

import (
	"kits-worker/kits/activities"
	"kits-worker/kits/models"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type KitsSolaceAPIWorkflowInput struct {
	KitsURL string           `json:"kitsURL"`
	Filter  string           `json:"filter"`
	Auth    models.BasicAuth `json:"solaceApiAuth"`
}

// KitsSolaceAPIWorkflow executes a one-off test against SolaceAPI for each kits and returns sempVersion
//
// Input:
//   - kitsURL: URL to the kits definition YAML.
//   - filter: Optional kit name filter. If empty, all kits are processed.
//   - solaceApiAuth: Basic authentication credentials for Solace SEMP APIs. This is for testing.
//     Those parameters are kit specific and should not be passed PLAIN? They should be fetched from Vault for specific kits (ie HCV).
//
// Example Temporal UI input:
//
//	{
//	  "kitsURL": "https://.../id-meshconfig-main_20260119_2.tar.gz",
//	  "filter": "fss-dce-sg-localtest1",
//	  "solaceApiAuth": {
//	    "Username": "admin",
//	    "Password": "admin"
//	  }
//	}
func KitsSolaceAPIWorkflow(
	ctx workflow.Context,
	input KitsSolaceAPIWorkflowInput,
) error {

	logger := workflow.GetLogger(ctx)
	logger.Info("Starting KitsSolaceAPIWorkflow", "kitsURL", input.KitsURL, "filter", input.Filter)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    5,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Download and parse kits
	var kits map[string]models.Kit
	if err := workflow.ExecuteActivity(
		ctx,
		activities.DownloadAndParseKitsActivity,
		input.KitsURL,
		input.Filter,
	).Get(ctx, &kits); err != nil {
		logger.Error("DownloadAndParseKitsActivity failed", "error", err)
		return err
	}

	// Execute child workflows
	var futures []workflow.ChildWorkflowFuture
	for name, kit := range kits {
		futures = append(futures,
			workflow.ExecuteChildWorkflow(ctx, KitSolaceAPIWorkflow, name, kit, input.Auth),
		)
	}

	// Wait for all children
	for _, f := range futures {
		if err := f.Get(ctx, nil); err != nil {
			logger.Error("Child workflow failed", "error", err)
			return err
		}
	}

	logger.Info("KitsSolaceAPIWorkflow completed successfully")
	return nil
}
