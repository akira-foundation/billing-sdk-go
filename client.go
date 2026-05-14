package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to the Akira Billing API on behalf of one product/customer pair.
type Client struct {
	BaseURL       string
	ProductSlug   string
	ProductSecret string
	CustomerToken string
	HTTP          *http.Client
}

// NewClient wires a default Client with a 10s timeout.
func NewClient(baseURL, productSlug, productSecret string) *Client {
	return &Client{
		BaseURL:       baseURL,
		ProductSlug:   productSlug,
		ProductSecret: productSecret,
		HTTP:          &http.Client{Timeout: 10 * time.Second},
	}
}

// SetCustomerToken stores the Bearer token the SDK should send on signed requests.
func (c *Client) SetCustomerToken(token string) {
	c.CustomerToken = token
}

// APIError represents a non-2xx response with the server-provided error code.
type APIError struct {
	Status int    `json:"-"`
	Code   string `json:"error"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("billing api %d: %s", e.Status, e.Code)
}

// Do builds, signs and dispatches a request, then decodes JSON into out (if non-nil).
// Pass an empty body slice for GET requests.
func (c *Client) Do(ctx context.Context, method, path string, body []byte, out any) error {
	endpoint, err := url.JoinPath(c.BaseURL, path)
	if err != nil {
		return fmt.Errorf("billing: build url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("billing: build request: %w", err)
	}

	nonce, err := NewNonce()
	if err != nil {
		return err
	}
	ts := time.Now().Unix()
	canonical := Canonical(c.ProductSlug, ts, nonce, method, "/"+strings.TrimLeft(path, "/"), body)

	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set(HeaderProduct, c.ProductSlug)
	req.Header.Set(HeaderTimestamp, fmt.Sprintf("%d", ts))
	req.Header.Set(HeaderNonce, nonce)
	req.Header.Set(HeaderSignature, Sign(c.ProductSecret, canonical))
	if c.CustomerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.CustomerToken)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("billing: do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		apiErr := &APIError{Status: resp.StatusCode}
		_ = json.Unmarshal(raw, apiErr)
		if apiErr.Code == "" {
			apiErr.Code = string(raw)
		}
		return apiErr
	}

	if out == nil {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(out)
}
