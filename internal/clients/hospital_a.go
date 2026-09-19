package clients

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var ErrPatientNotFound = errors.New("patient not found")

type HospitalAClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func (c *HospitalAClient) Search(ctx context.Context, id string) (json.RawMessage, error) {
	// Construct the request to Hospital A's HIS API
	endpoint := strings.TrimRight(c.BaseURL, "/") + "/patient/search/" + url.PathEscape(id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create Hospital A request: %w", err)
	}

	// Use the provided HTTP client or default to http.DefaultClient
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	// Send the request to Hospital A's HIS API
	response, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request Hospital A: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, ErrPatientNotFound
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Hospital A returned status %d", response.StatusCode)
	}

	// Read the response body with a size limit to prevent excessive memory usage
	body, err := io.ReadAll(io.LimitReader(response.Body, (2<<20)+1))
	if err != nil {
		return nil, fmt.Errorf("read Hospital A response: %w", err)
	}
	if len(body) > 2<<20 {
		return nil, fmt.Errorf("Hospital A response too large")
	}
	if !json.Valid(body) {
		return nil, fmt.Errorf("invalid Hospital A JSON response")
	}
	return json.RawMessage(body), nil
}
