// Package client provides the HMAC-signed transport for the Akira Billing API.
//
// The Client type owns the *http.Client, the product credentials, and the
// optional per-customer bearer. Endpoint helpers live in their own
// sub-packages (license, oauth, usage, ...) and take *client.Client as their
// first non-context argument.
package client

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

	"github.com/akira-io/billing-sdk-go/signature"
)

// Client talks to the Akira Billing API on behalf of one product/customer pair.
type Client struct {
	BaseURL       string
	ProductSlug   string
	ProductSecret string
	CustomerToken string
	HTTP          *http.Client
}

// New wires a default Client with a 10s timeout.
func New(baseURL, productSlug, productSecret string) *Client {
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
	Status  int    `json:"-"`
	Code    string `json:"error"`
	Message string `json:"message"`
}

func (e *APIError) reason() string {
	if e.Code != "" {
		return e.Code
	}
	return e.Message
}

func (e *APIError) Error() string {
	return fmt.Sprintf("billing api %d: %s", e.Status, e.reason())
}

// Do builds, signs and dispatches a request, then decodes JSON into out (if non-nil).
// Pass nil for body on GET requests.
func (c *Client) Do(ctx context.Context, method, path string, body []byte, out any) error {
	endpoint, err := url.JoinPath(c.BaseURL, path)
	if err != nil {
		return fmt.Errorf("billing: build url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("billing: build request: %w", err)
	}

	nonce, err := signature.NewNonce()
	if err != nil {
		return err
	}
	ts := time.Now().Unix()
	canonical := signature.Canonical(c.ProductSlug, ts, nonce, method, "/"+strings.TrimLeft(path, "/"), body)

	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set(signature.HeaderProduct, c.ProductSlug)
	req.Header.Set(signature.HeaderTimestamp, fmt.Sprintf("%d", ts))
	req.Header.Set(signature.HeaderNonce, nonce)
	req.Header.Set(signature.HeaderSignature, signature.Sign(c.ProductSecret, canonical))
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
		if apiErr.Code == "" && apiErr.Message == "" {
			apiErr.Code = string(raw)
		}
		return apiErr
	}

	if out == nil {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

// DoPublic dispatches a request without HMAC headers and without the customer
// bearer. Use only for endpoints documented as unauthenticated.
func (c *Client) DoPublic(ctx context.Context, method, path string, body []byte, out any) error {
	endpoint, err := url.JoinPath(c.BaseURL, path)
	if err != nil {
		return fmt.Errorf("billing: build url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("billing: build request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
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
		if apiErr.Code == "" && apiErr.Message == "" {
			apiErr.Code = string(raw)
		}
		return apiErr
	}

	if out == nil {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(out)
}
