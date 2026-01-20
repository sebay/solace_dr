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

func VerifyVPNRoleActivity(
	ctx context.Context,
	host string,
	port int,
	vpn string,
	role string,
	auth models.BasicAuth,
) error {

	url := fmt.Sprintf(
		"%s://%s:%d/SEMP/v2/monitor/msgVpns?where=enabled==true,replicationEnabled==true,replicationRole==%s,msgVpnName==%s",
		config.CURRENT_HTTP_SCHEME, host, port, role, vpn,
	)

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	applyBasicAuth(req, auth)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var parsed struct {
		Data []struct {
			Name string `json:"msgVpnName"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return err
	}

	if len(parsed.Data) == 0 {
		return fmt.Errorf("vpn %s not in %s role on %s", vpn, role, host)
	}

	return nil
}

func ValidateFinalRolesActivity(
	ctx context.Context,
	vpn string,
	oldActive models.MateResult,
	newActive models.MateResult,
	auth models.BasicAuth,
) error {

	if err := VerifyVPNRoleActivity(
		ctx,
		oldActive.Host,
		oldActive.Port,
		vpn,
		"standby",
		auth,
	); err != nil {
		return err
	}

	if err := VerifyVPNRoleActivity(
		ctx,
		newActive.Host,
		newActive.Port,
		vpn,
		"active",
		auth,
	); err != nil {
		return err
	}

	activity.GetLogger(ctx).Info(
		"vpn failover validated",
		"vpn", vpn,
	)

	return nil
}
