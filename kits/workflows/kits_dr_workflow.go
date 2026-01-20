package workflows

import (
	"kits-worker/kits/activities"
	"kits-worker/kits/models"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type KitsWorkflowInput struct {
	KitsURL string           `json:"kitsURL"`
	Filter  string           `json:"filter"`
	Auth    models.BasicAuth `json:"solaceApiAuth"`
}

// KitsDRWorkflow executes a one-off disaster recovery across kits.
//
// This workflow performs a single snapshot evaluation of the kits topology
// and broker state at the time of execution.
//
// Important notes:
//   - This is a one-off run; there is no automatic refresh of kitsURL.
//   - Long-running executions will NOT re-evaluate:
//   - kit topology changes
//   - broker state changes
//   - Any changes occurring after workflow start will not be detected.
//   - This DR Worklow listens for DNS vpn.<fqdn.com> (as defined for the moment in vpn_fqdn_config.go).
//     This means that any other fqdn update is not relevant to this process.
//
// Input:
//   - kitsURL: URL to the kits definition YAML.
//   - filter: Optional kit name filter. If empty, all kits are processed.
//   - solaceApiAuth: Basic authentication credentials for Solace SEMP APIs. This is for testing.
//     Those parameters are kit specific and should NOT be passed PLAIN! They should be fetched from Vault for specific kits (ie HCV).
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
func KitsDRWorkflow(
	ctx workflow.Context,
	input KitsWorkflowInput,
) ([]models.MateResult, error) {

	logger := workflow.GetLogger(ctx)
	logger.Info("Starting KitsDRWorkflow", "kitsURL", input.KitsURL, "filter", input.Filter)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second * 5,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    5, // <-- fail activity after 5 retries
			NonRetryableErrorTypes: []string{
				"DecodeError",  // XML decode errors won't retry
				"RequestError", // invalid request won't retry
			},
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
		return nil, err
	}

	// Execute child workflows
	var futures []workflow.ChildWorkflowFuture
	for name, kit := range kits {
		futures = append(futures,
			workflow.ExecuteChildWorkflow(ctx, KitDRWorkflow, name, kit, input.Auth),
		)
	}

	// Collect results
	var all []models.MateResult
	for _, f := range futures {
		var res []models.MateResult
		if err := f.Get(ctx, &res); err != nil {
			logger.Error("Child workflow failed", "error", err)
			return nil, err
		}
		all = append(all, res...)
	}

	logger.Info("KitsDRWorkflow completed successfully", "totalResults", len(all))
	return all, nil
}
