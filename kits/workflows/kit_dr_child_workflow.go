package workflows

import (
	"kits-worker/kits/activities"
	"kits-worker/kits/models"
	"time"

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

	// 1. Check mates on dc site
	var mates []workflow.Future
	check := func(dc, mate string, ep models.Endpoint) {
		mates = append(mates,
			workflow.ExecuteActivity(
				ctx,
				activities.CheckMateStatusActivity,
				kitName,
				dc,
				mate,
				ep,
				auth,
			),
		)
	}

	check("dc1", "mate1", kit.DC1.Mate1)
	check("dc1", "mate2", kit.DC1.Mate2)
	check("dc2", "mate1", kit.DC2.Mate1)
	check("dc2", "mate2", kit.DC2.Mate2)

	var active []models.MateResult
	for _, f := range mates {
		var r models.MateResult
		if err := f.Get(ctx, &r); err != nil {
			return nil, err
		}
		if r.Status == models.Active {
			active = append(active, r)
		}
	}

	vpnMapActive := make(map[string]models.MateResult)
	vpnMapStandby := make(map[string]models.MateResult)
	// 2. For each ACTIVE mate → get VPNs
	for _, m := range active {
		var vpns []string
		if err := workflow.ExecuteActivity(
			ctx,
			activities.GetRoleVPNsActivity,
			m.Host,
			m.Port,
			"active",
			auth,
		).Get(ctx, &vpns); err != nil {
			return nil, err
		}

		for _, vpn := range vpns {
			vpnMapActive[vpn] = m
		}

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

	// 3. For each VPN → start DNS watcher, passing vpn AND host/port where it is currently active and where it is standby
	var dnsFutures []workflow.ChildWorkflowFuture
	for vpn := range vpnMapActive {
		vpnCopy := vpn
		activeMate := vpnMapActive[vpnCopy]
		standbyMate := vpnMapStandby[vpnCopy]

		f := workflow.ExecuteChildWorkflow(ctx, VPNDNSWatchAndExecuteVPNFailoverWorkflow, vpnCopy, activeMate, standbyMate, auth)
		dnsFutures = append(dnsFutures, f)
	}

	// Optional: wait for all to start (or finish if you want)
	for _, f := range dnsFutures {
		// Just calling Get yields control; if DNS watchers run infinitely, consider using Get with timeout or skip Get entirely
		_ = f.Get(ctx, nil)
	}
	return active, nil
}
