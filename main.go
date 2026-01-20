package main

import (
	"kits-worker/kits/activities"
	"kits-worker/kits/workflows"
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	// 1 connect to Temporal server
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatal("unable to create Temporal client:", err)
	}
	defer c.Close()

	// 2 Create worker listening on "KITS_TASK_QUEUE"
	w := worker.New(c, "KITS_TASK_QUEUE", worker.Options{})

	// 3 Register workflows
	w.RegisterWorkflow(workflows.KitsDRWorkflow)
	w.RegisterWorkflow(workflows.KitDRWorkflow)
	w.RegisterWorkflow(workflows.VPNFailoverWorkflow)
	w.RegisterWorkflow(workflows.VPNDNSWatchAndExecuteVPNFailoverWorkflow)
	w.RegisterWorkflow(workflows.KitsSolaceAPIWorkflow)
	w.RegisterWorkflow(workflows.KitSolaceAPIWorkflow)

	// 4 Register activities
	w.RegisterActivity(activities.DownloadAndParseKitsActivity)
	w.RegisterActivity(activities.CheckMateStatusActivity)
	w.RegisterActivity(activities.GetRoleVPNsActivity)
	w.RegisterActivity(activities.ResolveDNSActivity)
	w.RegisterActivity(activities.SetVPNReplicationRoleActivity)
	w.RegisterActivity(activities.WaitForReplicationDrainActivity)
	w.RegisterActivity(activities.VerifyVPNRoleActivity)
	w.RegisterActivity(activities.ValidateFinalRolesActivity)
	w.RegisterActivity(activities.BrokerSEMPApiAboutActivity)

	log.Println("KITS worker started, listening on KITS_TASK_QUEUE...")

	// 5 Run worker
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatal("worker stopped with error:", err)
	}
}
