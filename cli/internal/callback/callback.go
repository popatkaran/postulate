// Package callback implements the local HTTP listener used during OAuth login.
// The listener binds to 127.0.0.1 on a random port in the range 18000–18099,
// receives the redirect from the Postulate API, and delivers the token data to
// the caller via a buffered channel — no shared memory.
package callback

import (
	"context"
	"fmt"
	"net"
	"net/http"
)

const (
	portRangeStart = 18000
	portRangeEnd   = 18099
)

// Result holds the token data extracted from the OAuth callback query parameters.
type Result struct {
	Token        string
	RefreshToken string
	ExpiresAt    string
	Role         string
	Err          error
}

// Server is a short-lived HTTP server that handles a single OAuth callback.
type Server struct {
	port     int
	result   chan Result
	ready    chan struct{}
	srv      *http.Server
	listener net.Listener
}

// New allocates a local listener on a random port in 18000–18099.
// Returns an error if no port in the range is available.
func New() (*Server, error) {
	for port := portRangeStart; port <= portRangeEnd; port++ {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			continue
		}
		return &Server{
			port:     port,
			result:   make(chan Result, 1),
			ready:    make(chan struct{}),
			listener: ln,
		}, nil
	}
	return nil, fmt.Errorf("no available port in range %d–%d", portRangeStart, portRangeEnd)
}

// Port returns the port the server will listen on.
func (s *Server) Port() int { return s.port }

// RedirectURI returns the full callback URL to pass to the OAuth initiation endpoint.
func (s *Server) RedirectURI() string {
	return fmt.Sprintf("http://127.0.0.1:%d/callback", s.port)
}

// Start begins listening and returns immediately. The server shuts down after
// the first callback or when ctx is cancelled (timeout).
// Results are delivered on the channel returned by C().
func (s *Server) Start(ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", s.handleCallback)
	s.srv = &http.Server{Handler: mux}

	go func() {
		close(s.ready) // signal that the listener is bound and ready
		_ = s.srv.Serve(s.listener)
	}()

	// Shut down when context is done (timeout or cancellation).
	go func() {
		<-ctx.Done()
		_ = s.srv.Shutdown(context.Background()) //nolint:contextcheck
		// If no result was delivered yet, send a timeout error.
		select {
		case s.result <- Result{Err: fmt.Errorf("login timed out — no callback received")}:
		default:
		}
	}()
}

// Ready returns a channel that is closed once the server is accepting connections.
func (s *Server) Ready() <-chan struct{} { return s.ready }

// C returns the channel on which the single Result will be delivered.
func (s *Server) C() <-chan Result { return s.result }

func (s *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	res := Result{
		Token:        q.Get("token"),
		RefreshToken: q.Get("refresh_token"),
		ExpiresAt:    q.Get("expires_at"),
		Role:         q.Get("role"),
	}
	if res.Token == "" {
		res.Err = fmt.Errorf("callback missing token parameter")
	}

	// Deliver result before responding so the channel is never blocked.
	select {
	case s.result <- res:
	default:
	}

	if res.Err != nil {
		http.Error(w, "Login failed. Return to your terminal.", http.StatusBadRequest)
	} else {
		fmt.Fprintln(w, "Login successful. You may close this tab.")
	}

	// Shut down the server asynchronously so this handler can return.
	go func() { _ = s.srv.Shutdown(context.Background()) }() //nolint:contextcheck
}
