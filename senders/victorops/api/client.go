package api

import "net/http"

// Client for the victorops API
type Client struct {
	httpClient *http.Client
	routingURL string
}

// NewClient returns a new victorops API client for the given routing URL
// and http client (Uses http.DefaultClient if httpClient is nil)
func NewClient(routingURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		httpClient: httpClient,
		routingURL: routingURL,
	}
}
