package devices

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/akira-io/billing-sdk-go/client"
)

func TestListReturnsPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/me/devices/spectra" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"slots_used":2,"slots_limit":3,"devices":[{"id":"d1","device_type":"desktop"}]}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "spectra", "secret")
	page, err := List(context.Background(), c, "spectra")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.SlotsUsed != 2 || page.SlotsLimit != 3 {
		t.Errorf("slots: got %d/%d", page.SlotsUsed, page.SlotsLimit)
	}
	if len(page.Devices) != 1 || page.Devices[0].ID != "d1" {
		t.Errorf("devices: %+v", page.Devices)
	}
}

func TestRevokeSendsDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/me/devices/d1" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "spectra", "secret")
	if err := Revoke(context.Background(), c, "d1"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
}

func TestLimitFromError(t *testing.T) {
	body := []byte(`{"code":"device_limit_reached","message":"limit","slots_limit":3,"slots_used":3,"devices":[{"id":"d1","device_type":"desktop"}]}`)
	apiErr := &client.APIError{Status: 409, Code: "device_limit_reached", Body: body}

	info, ok := LimitFromError(apiErr)
	if !ok {
		t.Fatal("expected limit error")
	}
	if info.SlotsLimit != 3 || info.SlotsUsed != 3 || len(info.Devices) != 1 {
		t.Errorf("info: %+v", info)
	}

	if _, ok := LimitFromError(&client.APIError{Status: 500, Code: "server_error"}); ok {
		t.Error("non-limit error should return false")
	}
}
