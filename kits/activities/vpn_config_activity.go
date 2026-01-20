package activities

import (
	"bytes"
	"context"
	"fmt"
	"kits-worker/kits/config"
	"kits-worker/kits/models"
	"net/http"

	"go.temporal.io/sdk/activity"
)

func SetVPNReplicationRoleActivity(
	ctx context.Context,
	host string,
	port int,
	vpn string,
	role string,
	auth models.BasicAuth,
) error {

	url := fmt.Sprintf(
		"%s://%s:%d/SEMP/v2/config/msgVpns/%s",
		config.CURRENT_HTTP_SCHEME,
		host, port, vpn,
	)

	body := []byte(fmt.Sprintf(`{"replicationRole":"%s"}`, role))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPatch,
		url,
		bytes.NewBuffer(body),
	)
	if err != nil {
		return err
	}

	applyBasicAuth(req, auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	activity.GetLogger(ctx).Info(
		"vpn role updated",
		"vpn", vpn,
		"host", host,
		"role", role,
	)

	return nil
}
