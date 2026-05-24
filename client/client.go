package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/akira-io/billing-sdk-go/signature"
)

type Client struct {
	BaseURL       string
	ProductSlug   string
	ProductSecret string
	CustomerToken string
	HTTP          *http.Client
}

func New(baseURL, productSlug, productSecret string) *Client {
	return &Client{
		BaseURL:       baseURL,
		ProductSlug:   productSlug,
		ProductSecret: productSecret,
		HTTP: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, _ []*http.Request) error {
				return fmt.Errorf("billing: unexpected redirect to %s", req.URL)
			},
		},
	}
}

func (c *Client) SetCustomerToken(token string) {
	c.CustomerToken = token
}

type APIError struct {
	Status     int    `json:"-"`
	Code       string `json:"error"`
	Message    string `json:"message"`
	RetryAfter int    `json:"-"`
}

func parseRetryAfter(h http.Header) int {
	secs, err := strconv.Atoi(strings.TrimSpace(h.Get("Retry-After")))
	if err != nil {
		return 0
	}
	return secs
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
		apiErr := &APIError{Status: resp.StatusCode, RetryAfter: parseRetryAfter(resp.Header)}
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
		apiErr := &APIError{Status: resp.StatusCode, RetryAfter: parseRetryAfter(resp.Header)}
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
