package healthcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Client represents a client for fetching healthcheck states from the target url
type Client struct {
	URL    string
	Client *http.Client
}

// Check returns the healthcheck states from the target
func (c Client) Check(ctx context.Context) (*CheckingResults, error) {
	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}

	request, err := http.NewRequest(http.MethodGet, c.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create a request for the target: %w", err)
	}

	request = request.WithContext(ctx)
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to get a response from the target '%s': %w", c.URL, err)
	}

	if code := response.StatusCode; code < 200 || code >= 300 {
		return nil, fmt.Errorf("unexpected response status HTTP %d: %s", code, response.Status)
	}

	var results CheckingResults
	defer response.Body.Close()
	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read the response from the target '%s': %w", c.URL, err)
	}

	if err := json.Unmarshal(b, &results); err != nil {
		return nil, fmt.Errorf("failed to json-unmarshal the response from the target '%s': %s", c.URL, string(b))
	}

	return &results, nil
}
