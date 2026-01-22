package workflows

import (
	"kits-worker/kits/activities"
	"kits-worker/kits/models"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func VPNFailoverWorkflow(
	ctx workflow.Context,
	vpn string,
	active *models.MateResult,
	standby *models.MateResult,
	auth models.BasicAuth,
) error {

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// 1) ACTIVE → STANDBY
	if active != nil {
		if err := workflow.ExecuteActivity(
			ctx,
			activities.SetVPNReplicationRoleActivity,
			active.Host,
			active.Port,
			vpn,
			"standby",
			auth,
		).Get(ctx, nil); err != nil {
			return err
		}

		// 2) Monitor replication queue (12 retries, 5s)
		if err := workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: 70 * time.Second,
				RetryPolicy: &temporal.RetryPolicy{
					MaximumAttempts: 12,
					InitialInterval: 5 * time.Second,
				},
			}),
			activities.WaitForReplicationDrainActivity,
			active.Host,
			active.Port,
			vpn,
			auth,
		).Get(ctx, nil); err != nil {
			return err
		}
	} else {
		workflow.GetLogger(ctx).Warn(
			"skipping active site as not present",
		)
	}

	// 3) Verify VPN is standby on other DC
	if err := workflow.ExecuteActivity(
		ctx,
		activities.VerifyVPNRoleActivity,
		standby.Host,
		standby.Port,
		vpn,
		"standby",
		auth,
	).Get(ctx, nil); err != nil {
		return err
	}

	// 4) STANDBY → ACTIVE (other DC)
	if err := workflow.ExecuteActivity(
		ctx,
		activities.SetVPNReplicationRoleActivity,
		standby.Host,
		standby.Port,
		vpn,
		"active",
		auth,
	).Get(ctx, nil); err != nil {
		return err
	}

	// 5) Final validation that vpn is standby on previously main dc and is now active on previously standby dc
	if err := workflow.ExecuteActivity(
		ctx,
		activities.ValidateFinalRolesActivity,
		vpn,
		active,
		standby,
		auth,
	).Get(ctx, nil); err != nil {
		return err
	}

	//todo check if any error and return it as failed

	return nil
}
