package workflows

import (
	"kits-worker/kits/activities"
	"kits-worker/kits/config"
	"kits-worker/kits/models"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func VPNDNSWatchAndExecuteVPNFailoverWorkflow(ctx workflow.Context, vpn string, active models.MateResult, standby models.MateResult, auth models.BasicAuth) error {
	logger := workflow.GetLogger(ctx)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:    5,
			InitialInterval:    time.Second * 5,
			BackoffCoefficient: 2.0,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	dnsNameA := vpn + config.VPN_PRIMARY_BROKER_FQDN_GLOBAL_DOMAIN
	dnsNameB := vpn + config.VPN_BACKUP_BROKER_FQDN_GLOBAL_DOMAIN

	var lastIPA, lastIPB string

	for {
		var currentIPA, currentIPB string

		// Resolve first DNS
		if err := workflow.ExecuteActivity(ctx, activities.ResolveDNSActivity, dnsNameA).Get(ctx, &currentIPA); err != nil {
			logger.Error("DNS resolve failed", "dnsName", dnsNameA, "error", err)
			workflow.Sleep(ctx, 10*time.Second)
			continue
		}
		logger.Info("DNS resolved", "dnsName", dnsNameA, "ip", currentIPA)

		// Resolve second DNS
		if err := workflow.ExecuteActivity(ctx, activities.ResolveDNSActivity, dnsNameB).Get(ctx, &currentIPB); err != nil {
			logger.Error("DNS resolve failed", "dnsName", dnsNameB, "error", err)
			workflow.Sleep(ctx, 10*time.Second)
			continue
		}
		logger.Info("DNS resolved", "dnsName", dnsNameB, "ip", currentIPB)

		// Check if BOTH DNS IPs changed
		dnsAChanged := lastIPA != "" && currentIPA != lastIPA
		dnsBChanged := lastIPB != "" && currentIPB != lastIPB

		if dnsAChanged || dnsBChanged {
			logger.Info("DNS changes detected", "vpn", vpn, "dnsAChanged", dnsAChanged, "dnsBChanged", dnsBChanged)
		}

		dnsChanged := dnsAChanged && dnsBChanged

		if dnsChanged {
			if err := workflow.ExecuteChildWorkflow(ctx, VPNFailoverWorkflow, vpn, active, standby, auth).Get(ctx, nil); err != nil {
				return err
			}

			logger.Info("VPN failover completed, stopping DNS watcher")
			break
		}

		lastIPA = currentIPA
		lastIPB = currentIPB

		workflow.Sleep(ctx, 10*time.Second)
	}

	return nil
}
