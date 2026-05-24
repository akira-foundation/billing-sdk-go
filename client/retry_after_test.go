package client

import (
	"net/http"
	"testing"
)

func TestParseRetryAfter(t *testing.T) {
	seconds := http.Header{}
	seconds.Set("Retry-After", "30")
	if got := parseRetryAfter(seconds); got != 30 {
		t.Fatalf("seconds: want 30, got %d", got)
	}

	if got := parseRetryAfter(http.Header{}); got != 0 {
		t.Fatalf("absent: want 0, got %d", got)
	}

	httpDate := http.Header{}
	httpDate.Set("Retry-After", "Wed, 21 Oct 2026 07:28:00 GMT")
	if got := parseRetryAfter(httpDate); got != 0 {
		t.Fatalf("http-date: want 0, got %d", got)
	}
}
