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

type vpnResponse struct {
	Data []struct {
		Name string `json:"msgVpnName"`
	} `json:"data"`
}

func GetRoleVPNsActivity(
	ctx context.Context,
	host string,
	port int,
	role string,
	auth models.BasicAuth,
) ([]string, error) {

	url := fmt.Sprintf(
		"%s://%s:%d/SEMP/v2/monitor/msgVpns?where=enabled==true,replicationEnabled==true,replicationRole==%s,msgVpnName!=#*",
		config.CURRENT_HTTP_SCHEME,
		host,
		port,
		role,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	applyBasicAuth(req, auth)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed vpnResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	var vpns []string
	for _, d := range parsed.Data {
		vpns = append(vpns, d.Name)
	}

	activity.GetLogger(ctx).Info(
		"VPNs discovered",
		"role", role,
		"count", len(vpns),
	)

	return vpns, nil
}
