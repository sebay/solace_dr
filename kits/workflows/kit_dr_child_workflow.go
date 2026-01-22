package workflows

import (
	"errors"
	"time"

	"kits-worker/kits/activities"
	"kits-worker/kits/models"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func KitDRWorkflow(
	ctx workflow.Context,
	kitName string,
	kit models.Kit,
	auth models.BasicAuth,
) ([]models.MateResult, error) {

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	checkAO := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1, // ⬅️ CRITICAL
		},
	}

	checkCtx := workflow.WithActivityOptions(ctx, checkAO)

	/*
		1. Check mates (retry 3 times, ignore failure)
	*/
	type mateCheck struct {
		dc   string
		mate string
		ep   models.Endpoint
	}

	mateChecks := []mateCheck{
		{"dc1", "mate1", kit.DC1.Mate1},
		{"dc1", "mate2", kit.DC1.Mate2},
		{"dc2", "mate1", kit.DC2.Mate1},
		{"dc2", "mate2", kit.DC2.Mate2},
	}

	var active []models.MateResult

	for _, mc := range mateChecks {
		var r models.MateResult
		err := workflow.ExecuteActivity(
			checkCtx,
			activities.CheckMateStatusActivity,
			kitName,
			mc.dc,
			mc.mate,
			mc.ep,
			auth,
		).Get(checkCtx, &r)

		if err == nil && r.Status == models.Active {
			active = append(active, r)
		}
	}

	if len(active) == 0 {
		return nil, errors.New("no active mates found after 3 retries")
	}

	has2ActiveMate := len(active) == 2

	/*
		2. Build VPN maps from ACTIVE mates
	*/
	vpnMapActive := make(map[string]models.MateResult)
	vpnMapStandby := make(map[string]models.MateResult)

	for _, m := range active {
		// Active VPNs
		var vpnsActive []string
		if err := workflow.ExecuteActivity(
			ctx,
			activities.GetRoleVPNsActivity,
			m.Host,
			m.Port,
			"active",
			auth,
		).Get(ctx, &vpnsActive); err != nil {
			return nil, err
		}

		for _, vpn := range vpnsActive {
			vpnMapActive[vpn] = m
		}

		// Standby VPNs
		var vpnsStandby []string
		if err := workflow.ExecuteActivity(
			ctx,
			activities.GetRoleVPNsActivity,
			m.Host,
			m.Port,
			"standby",
			auth,
		).Get(ctx, &vpnsStandby); err != nil {
			return nil, err
		}

		for _, vpn := range vpnsStandby {
			vpnMapStandby[vpn] = m
		}
	}

	/*
		3. Start DNS watcher child workflows
	*/
	var dnsFutures []workflow.ChildWorkflowFuture

	if has2ActiveMate {
		// Normal path: VPNs from active map
		for vpn := range vpnMapActive {
			vpnCopy := vpn
			m := vpnMapActive[vpnCopy]
			activeMate := &m
			standbyMate := vpnMapStandby[vpnCopy]

			f := workflow.ExecuteChildWorkflow(
				ctx,
				VPNDNSWatchAndExecuteVPNFailoverWorkflow,
				vpnCopy,
				activeMate,
				standbyMate,
				auth,
			)
			dnsFutures = append(dnsFutures, f)
		}
	} else {
		if len(vpnMapStandby) == 0 {
			workflow.GetLogger(ctx).Warn(
				"There is 1 active mate on active site but standby is not responding. " +
					"We have no site to failover to. Skipping DR.",
			)
		} else {
			workflow.GetLogger(ctx).Warn(
				"There is 1 active mate on standby site but primary site is unreachable. " +
					"Proceeding with activating Standby Site without access to Active Site",
			)
			// No active mates at all → VPNs from standby map
			for vpn := range vpnMapStandby {
				vpnCopy := vpn
				var activeMate *models.MateResult = nil
				standbyMate := vpnMapStandby[vpnCopy]

				f := workflow.ExecuteChildWorkflow(
					ctx,
					VPNDNSWatchAndExecuteVPNFailoverWorkflow,
					vpnCopy,
					activeMate,
					standbyMate,
					auth,
				)
				dnsFutures = append(dnsFutures, f)
			}
		}
	}

	/*
		4. Optional wait (safe even for long-running children)
	*/
	for _, f := range dnsFutures {
		_ = f.Get(ctx, nil)
	}

	return active, nil
}
