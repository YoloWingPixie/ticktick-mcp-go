package ticktick

import "net/http"

// NewClientWithBaseURL creates a Client with a custom base URL. Used for testing.
func NewClientWithBaseURL(httpClient *http.Client, baseURL, version string) *Client {
	return &Client{http: httpClient, baseURL: baseURL, userAgent: "ticktick-mcp-go/" + version}
}
