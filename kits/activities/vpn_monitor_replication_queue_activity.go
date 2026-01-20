package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"kits-worker/kits/config"
	"kits-worker/kits/models"
	"net/http"

	"go.temporal.io/sdk/activity"
)

type queueResponse struct {
	Data []struct {
		QueueName string  `json:"queueName"`
		TxMsgRate float64 `json:"txMsgRate"`
	} `json:"data"`
}

func WaitForReplicationDrainActivity(
	ctx context.Context,
	host string,
	port int,
	vpn string,
	auth models.BasicAuth,
) error {

	url := fmt.Sprintf(
		"%s://%s:%d/SEMP/v2/monitor/msgVpns/%s/queues?select=queueName,txMsgRate&where=queueName==%%23MSGVPN_REPLICATION_DATA_QUEUE,txMsgRate>0",
		config.CURRENT_HTTP_SCHEME, host, port, vpn,
	)

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	applyBasicAuth(req, auth)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var r queueResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return err
	}

	if len(r.Data) > 0 {
		return fmt.Errorf("replication still in progress")
	}

	activity.GetLogger(ctx).Info("replication queue drained", "vpn", vpn)
	return nil
}
