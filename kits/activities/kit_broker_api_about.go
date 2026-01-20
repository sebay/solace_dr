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

// AboutResponse mirrors the SEMP /monitor/about/api JSON response
type AboutResponse struct {
	Platform    string `json:"platform"`
	SempVersion string `json:"sempVersion"`
}

// aboutWrapper is the generic wrapper returned by /monitor/about and /monitor/about/api
type aboutWrapper struct {
	Data  map[string]interface{} `json:"data"`
	Links struct {
		APIUri string `json:"apiUri"`
		URI    string `json:"uri"`
	} `json:"links"`
	Meta struct {
		Request struct {
			Method string `json:"method"`
			URI    string `json:"uri"`
		} `json:"request"`
		ResponseCode int `json:"responseCode"`
	} `json:"meta"`
}

func BrokerSEMPApiAboutActivity(
	ctx context.Context,
	kit string,
	dc string,
	mate string,
	ep models.Endpoint,
	auth models.BasicAuth,
) (*AboutResponse, error) {

	logger := activity.GetLogger(ctx)

	// Step 1: call /monitor/about to get links.apiUri
	aboutURL := fmt.Sprintf(
		"%s://%s:%d/SEMP/v2/monitor/about",
		config.CURRENT_HTTP_SCHEME,
		ep.Host,
		ep.Port,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, aboutURL, nil)
	if err != nil {
		return nil, err
	}
	applyBasicAuth(req, auth)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var wrapper aboutWrapper
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, err
	}

	logger.Info(
		"SEMP /about retrieved",
		"kit", kit,
		"dc", dc,
		"mate", mate,
		"host", ep.Host,
		"port", ep.Port,
		"apiUri", wrapper.Links.APIUri,
		"responseCode", wrapper.Meta.ResponseCode,
	)

	// Step 2: call links.apiUri to get real platform/sempVersion
	apiReq, err := http.NewRequestWithContext(ctx, http.MethodGet, wrapper.Links.APIUri, nil)
	if err != nil {
		return nil, err
	}
	applyBasicAuth(apiReq, auth)

	apiResp, err := http.DefaultClient.Do(apiReq)
	if err != nil {
		return nil, err
	}
	defer apiResp.Body.Close()

	var apiWrapper aboutWrapper
	if err := json.NewDecoder(apiResp.Body).Decode(&apiWrapper); err != nil {
		return nil, err
	}

	// Extract platform and sempVersion from data
	data := apiWrapper.Data
	platform, _ := data["platform"].(string)
	sempVersion, _ := data["sempVersion"].(string)

	logger.Info(
		"SEMP /about/api retrieved",
		"kit", kit,
		"dc", dc,
		"mate", mate,
		"host", ep.Host,
		"port", ep.Port,
		"platform", platform,
		"sempVersion", sempVersion,
	)

	return &AboutResponse{
		Platform:    platform,
		SempVersion: sempVersion,
	}, nil
}
