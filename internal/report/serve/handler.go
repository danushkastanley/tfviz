package serve

import (
	"context"
	"net/http"
	"time"
)

// Handler applies the explorer's protections to every request:
//   - only the loopback host names this server is bound to are accepted;
//   - every response is no-store, unframeable and carries no CORS headers,
//     so other sites can neither read the page nor its data;
//   - the page needs the session cookie, obtained by opening the printed
//     link, and the cookie is SameSite=Strict and HttpOnly;
//   - refresh additionally needs a same-origin Origin header and the page's
//     CSRF token.
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Cache-Control", "no-store")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Content-Security-Policy", "frame-ancestors 'none'")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")

		s.mu.RLock()
		hostOK := s.hosts[r.Host]
		s.mu.RUnlock()
		if !hostOK {
			http.Error(w, "This explorer only answers on its own loopback address.", http.StatusMisdirectedRequest)
			return
		}
		switch {
		case r.URL.Path == "/" && r.Method == http.MethodGet:
			s.servePage(w, r)
		case r.URL.Path == refreshPath && r.Method == http.MethodPost:
			s.refresh(w, r)
		case r.URL.Path == "/" || r.URL.Path == refreshPath:
			http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		default:
			http.NotFound(w, r)
		}
	})
}

func (s *Server) servePage(w http.ResponseWriter, r *http.Request) {
	if t := r.URL.Query().Get("token"); t != "" {
		if !equal(t, s.access) {
			http.Error(w, "This link is not valid. Open the link printed by tfviz explore.", http.StatusForbidden)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: cookieName, Value: s.access, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
		// Drop the token from the address bar and history.
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !s.authenticated(r) {
		http.Error(w, "Open the link printed by tfviz explore.", http.StatusForbidden)
		return
	}
	s.mu.RLock()
	page := s.current
	s.mu.RUnlock()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(page)
}

// refresh re-reads the snapshot. It runs only on an explicit user action.
func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if !s.authenticated(r) || !equal(r.Header.Get(tokenHeader), s.csrf) || origin != "http://"+r.Host {
		http.Error(w, "Refresh is only available from the explorer page.", http.StatusForbidden)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	if err := s.reload(ctx); err != nil {
		// The message is tfviz's own guidance; it never contains input content.
		http.Error(w, "The snapshot could not be read again: "+err.Error(), http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) authenticated(r *http.Request) bool {
	c, err := r.Cookie(cookieName)
	return err == nil && equal(c.Value, s.access)
}
