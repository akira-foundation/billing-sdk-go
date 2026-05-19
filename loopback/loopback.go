// Package loopback drives the desktop loopback PKCE OAuth flow:
// binds a transient 127.0.0.1 listener, opens the system browser,
// awaits the provider callback, and exchanges the code via the SDK.
package loopback

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/akira-io/billing-sdk-go/client"
	"github.com/akira-io/billing-sdk-go/oauth"
)

const (
	defaultLoopbackTimeout = 5 * time.Minute
	loopbackReadTimeout    = 15 * time.Second
)

const loopbackSuccessHTML = `<!doctype html><meta charset=utf-8><title>Sign in complete</title><style>body{font-family:-apple-system,system-ui,sans-serif;background:#08080b;color:#e6e6ec;display:grid;place-items:center;height:100vh;margin:0}</style><h1>You can close this tab.</h1>`

// BrowserOpener launches the system default browser at url.
type BrowserOpener func(url string) error

// Options configures Login.
type Options struct {
	Provider string
	Product  string
	Timeout  time.Duration
}

// Outcome carries the exchange result returned by oauth.Exchange.
type Outcome struct {
	Exchange oauth.ExchangeResponse
}

// Login runs the desktop loopback PKCE OAuth flow end-to-end:
//
//  1. Binds a transient 127.0.0.1 listener.
//  2. Generates PKCE + state, builds the provider URL.
//  3. Calls openBrowser(url).
//  4. Awaits the callback (default 5 min).
//  5. Exchanges the code via oauth.Exchange.
//  6. Stores the bearer on c.
func Login(ctx context.Context, c *client.Client, opts Options, openBrowser BrowserOpener) (*Outcome, error) {
	if opts.Provider == "" {
		return nil, errors.New("billing: provider required")
	}
	if opts.Product == "" {
		return nil, errors.New("billing: product required")
	}
	if openBrowser == nil {
		return nil, errors.New("billing: open_browser required")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("billing: bind callback: %w", err)
	}
	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return nil, errors.New("billing: unexpected listener addr")
	}
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/cb", addr.Port)

	pkce, err := oauth.GeneratePkceChallenge()
	if err != nil {
		return nil, fmt.Errorf("billing: pkce: %w", err)
	}
	state, err := oauth.GenerateState()
	if err != nil {
		return nil, fmt.Errorf("billing: state: %w", err)
	}

	authURL := oauth.BuildInitURL(oauth.InitURLOptions{
		BaseURL:             c.BaseURL,
		Provider:            opts.Provider,
		Product:             opts.Product,
		RedirectURI:         redirectURI,
		CodeChallenge:       pkce.Challenge,
		CodeChallengeMethod: pkce.Method,
		State:               state,
	})

	if err := openBrowser(authURL); err != nil {
		return nil, fmt.Errorf("billing: open browser: %w", err)
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultLoopbackTimeout
	}
	callbackCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	code, returnedState, err := acceptCallback(callbackCtx, listener)
	if err != nil {
		return nil, err
	}
	if returnedState != state {
		return nil, errors.New("billing: oauth state mismatch")
	}

	exchange, err := oauth.Exchange(ctx, c, oauth.ExchangePayload{
		Code:         code,
		CodeVerifier: pkce.Verifier,
	})
	if err != nil {
		return nil, fmt.Errorf("billing: exchange code: %w", err)
	}

	return &Outcome{Exchange: *exchange}, nil
}

func acceptCallback(ctx context.Context, listener net.Listener) (string, string, error) {
	type result struct {
		conn net.Conn
		err  error
	}
	resultCh := make(chan result, 1)
	go func() {
		conn, err := listener.Accept()
		resultCh <- result{conn: conn, err: err}
	}()

	var conn net.Conn
	select {
	case <-ctx.Done():
		_ = listener.Close()
		return "", "", fmt.Errorf("billing: oauth callback: %w", ctx.Err())
	case r := <-resultCh:
		if r.err != nil {
			return "", "", fmt.Errorf("billing: accept callback: %w", r.err)
		}
		conn = r.conn
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(loopbackReadTimeout))

	reader := bufio.NewReader(conn)
	requestLine, err := reader.ReadString('\n')
	if err != nil {
		return "", "", fmt.Errorf("billing: read request line: %w", err)
	}
	parts := strings.Fields(requestLine)
	if len(parts) < 2 {
		return "", "", errors.New("billing: malformed request line")
	}

	requestURL, err := url.Parse(parts[1])
	if err != nil {
		return "", "", fmt.Errorf("billing: parse url: %w", err)
	}
	code := requestURL.Query().Get("code")
	state := requestURL.Query().Get("state")

	resp := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\nConnection: close\r\nContent-Length: %d\r\n\r\n%s",
		len(loopbackSuccessHTML),
		loopbackSuccessHTML,
	)
	_, _ = conn.Write([]byte(resp))

	if code == "" {
		return "", "", errors.New("billing: callback missing code")
	}
	if state == "" {
		return "", "", errors.New("billing: callback missing state")
	}

	return code, state, nil
}
