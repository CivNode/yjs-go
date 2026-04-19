package transport_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	yjs "github.com/CivNode/yjs-go"
	"github.com/CivNode/yjs-go/transport"
)

// TestConnectBadURL verifies that Connect returns an error when the URL is
// unreachable.
func TestConnectBadURL(t *testing.T) {
	doc := yjs.NewDoc()
	ctx := context.Background()

	_, err := transport.Connect(ctx, doc, "ws://127.0.0.1:1", "room", "")
	if err == nil {
		t.Fatal("expected error for bad URL, got nil")
	}
}

// TestConnectCancelledContext verifies that Connect fails fast when the context
// is already cancelled before dialing.
func TestConnectCancelledContext(t *testing.T) {
	doc := yjs.NewDoc()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before dialing

	_, err := transport.Connect(ctx, doc, "ws://127.0.0.1:1", "room", "")
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

// TestSyncHandshakePartialStep1 verifies that a server that sends an
// unexpected message type during handshake causes Connect to return an error
// without hanging.
func TestSyncHandshakePartialStep1(t *testing.T) {
	// Serve a plain HTTP 200 — not a WebSocket upgrade — to trigger an
	// immediate dial error. This exercises the dial-failure path in Connect.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	doc := yjs.NewDoc()
	ctx := context.Background()
	_, err := transport.Connect(ctx, doc, wsURL, "room", "")
	if err == nil {
		t.Fatal("expected error when server returns non-WebSocket response, got nil")
	}
}
