package activities

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"kits-worker/kits/config"
	"kits-worker/kits/models"
	"net/http"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

const sempPayload = `<rpc><show><redundancy/></show></rpc>`

type sempReply struct {
	RPC struct {
		Show struct {
			Redundancy struct {
				VirtualRouters struct {
					Primary *routerStatus `xml:"primary"`
					Backup  *routerStatus `xml:"backup"`
				} `xml:"virtual-routers"`
			} `xml:"redundancy"`
		} `xml:"show"`
	} `xml:"rpc"`
}

type routerStatus struct {
	Status struct {
		Activity string `xml:"activity"`
	} `xml:"status"`
}

func CheckMateStatusActivity(
	ctx context.Context,
	kit string,
	dc string,
	mate string,
	ep models.Endpoint,
	auth models.BasicAuth,
) (models.MateResult, error) {

	url := fmt.Sprintf("%s://%s:%d/SEMP", config.CURRENT_HTTP_SCHEME, ep.Host, ep.Port)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewBufferString(sempPayload),
	)
	if err != nil {
		return models.MateResult{}, temporal.NewNonRetryableApplicationError(
			"failed to create request",
			"RequestError",
			err,
		)
	}

	applyBasicAuth(req, auth)
	req.Header.Set("Content-Type", "application/xml")

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		// DNS, timeout, connection refused → retryable
		return models.MateResult{}, err
	}
	defer resp.Body.Close()

	var reply sempReply
	if err := xml.NewDecoder(resp.Body).Decode(&reply); err != nil {
		return models.MateResult{}, temporal.NewNonRetryableApplicationError(
			"failed to decode SEMP XML",
			"DecodeError",
			err,
		)
	}

	status := models.Standby

	vr := reply.RPC.Show.Redundancy.VirtualRouters
	if vr.Primary != nil && vr.Primary.Status.Activity == "Local Active" {
		status = models.Active
	}
	if vr.Backup != nil && vr.Backup.Status.Activity == "Local Active" {
		status = models.Active
	}

	activity.GetLogger(ctx).Info(
		"mate checked",
		"kit", kit,
		"dc", dc,
		"mate", mate,
		"host", ep.Host,
		"port", ep.Port,
		"status", status,
	)

	return models.MateResult{
		Kit:    kit,
		DC:     dc,
		Mate:   mate,
		Host:   ep.Host,
		Port:   ep.Port,
		Status: status,
	}, nil
}
