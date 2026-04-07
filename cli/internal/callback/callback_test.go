package callback_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/popatkaran/postulate/cli/internal/callback"
)

func TestNew_AllocatesPort(t *testing.T) {
	srv, err := callback.New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if srv.Port() < 18000 || srv.Port() > 18099 {
		t.Errorf("port %d out of expected range 18000–18099", srv.Port())
	}
}

func TestRedirectURI_Format(t *testing.T) {
	srv, err := callback.New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	expected := fmt.Sprintf("http://127.0.0.1:%d/callback", srv.Port())
	if srv.RedirectURI() != expected {
		t.Errorf("expected %q, got %q", expected, srv.RedirectURI())
	}
}

func TestCallback_HappyPath_DeliversResult(t *testing.T) {
	srv, err := callback.New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Start(ctx)
	<-srv.Ready()

	// Simulate the API redirect.
	callbackURL := fmt.Sprintf("%s?token=jwt123&refresh_token=rt456&expires_at=2099-01-01T00:00:00Z&role=platform_member",
		srv.RedirectURI())
	resp, err := http.Get(callbackURL) //nolint:noctx
	if err != nil {
		t.Fatalf("GET callback: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	select {
	case res := <-srv.C():
		if res.Err != nil {
			t.Fatalf("unexpected error: %v", res.Err)
		}
		if res.Token != "jwt123" {
			t.Errorf("token: expected jwt123, got %q", res.Token)
		}
		if res.RefreshToken != "rt456" {
			t.Errorf("refresh_token: expected rt456, got %q", res.RefreshToken)
		}
		if res.Role != "platform_member" {
			t.Errorf("role: expected platform_member, got %q", res.Role)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for callback result")
	}
}

func TestCallback_Timeout_DeliversError(t *testing.T) {
	srv, err := callback.New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	srv.Start(ctx)

	select {
	case res := <-srv.C():
		if res.Err == nil {
			t.Error("expected timeout error, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for timeout result")
	}
}

func TestCallback_MissingToken_DeliversError(t *testing.T) {
	srv, err := callback.New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Start(ctx)
	<-srv.Ready()

	// Callback with no token param.
	resp, err := http.Get(srv.RedirectURI()) //nolint:noctx
	if err != nil {
		t.Fatalf("GET callback: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	select {
	case res := <-srv.C():
		if res.Err == nil {
			t.Error("expected error for missing token, got nil")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for result")
	}
}
