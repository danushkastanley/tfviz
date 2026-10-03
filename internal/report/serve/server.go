// Package serve runs the local explorer: the report interface served from
// a loopback-only HTTP server (plan §11). It exposes only the projected
// report, never raw input, files or AWS operations.
package serve

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/danushkastanley/tfviz/internal/report/html"
	"github.com/danushkastanley/tfviz/internal/report/model"
)

const (
	cookieName  = "tfviz_session"
	refreshPath = "/refresh"
	tokenHeader = "X-Tfviz-Token"
)

// Loader reads the snapshot again and builds a report from it.
type Loader func(ctx context.Context) (*model.Report, error)

// Server holds the current page and the per-run secrets.
type Server struct {
	load   Loader
	assets html.Assets
	access string // in the printed URL; exchanged for a cookie
	csrf   string // embedded in the page; required on refresh

	mu      sync.RWMutex
	current []byte
	hosts   map[string]bool
}

// New builds the first page. It does not listen yet.
func New(ctx context.Context, load Loader, assets html.Assets) (*Server, error) {
	s := &Server{load: load, assets: assets, access: token(), csrf: token()}
	if err := s.reload(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

// Listen binds to the loopback interface only. Port 0 picks a free port.
func (s *Server) Listen(port int) (net.Listener, error) {
	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return nil, fmt.Errorf("cannot listen on 127.0.0.1:%d: %w", port, err)
	}
	s.AllowHost(ln.Addr().String())
	return ln, nil
}

// AllowHost records the address browsers will use. Requests naming any
// other host are refused, which defeats DNS rebinding.
func (s *Server) AllowHost(addr string) {
	_, port, _ := net.SplitHostPort(addr)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hosts = map[string]bool{"127.0.0.1:" + port: true, "localhost:" + port: true}
}

// URL is the link to open, carrying the one-time access token.
func (s *Server) URL(addr string) string {
	return "http://" + addr + "/?token=" + s.access
}

// Serve runs until ctx is cancelled, then shuts down gracefully.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	srv := &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	errs := make(chan error, 1)
	go func() { errs <- srv.Serve(ln) }()
	select {
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdown)
	case err := <-errs:
		return err
	}
}

func (s *Server) reload(ctx context.Context) error {
	report, err := s.load(ctx)
	if err != nil {
		return err
	}
	var page bytes.Buffer
	if err := html.RenderExplore(&page, report, s.assets, html.Explore{RefreshPath: refreshPath, CSRFToken: s.csrf}); err != nil {
		return err
	}
	s.mu.Lock()
	s.current = page.Bytes()
	s.mu.Unlock()
	return nil
}

func token() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func equal(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
