// Package server wires redline's HTTP surface.
//
// Every handler here is a plain http.Handler registered on a stdlib mux and
// every write goes through packet.Store. That is deliberate: the same
// handlers are meant to mount inside the existing Go Lambda service later,
// with an S3-backed store, without touching this file.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"redline/internal/packet"
	"redline/internal/snapshot"
)

const maxJSONBytes = 2 << 20

// Snapshot is one frozen document held in memory for the life of the process.
// It carries the session bookkeeping that tells the frontend whether this is
// round 1 of something new or round N+1 of an ongoing conversation.
type Snapshot struct {
	ID          string
	HTML        string
	Source      packet.Source
	SessionID   string
	ParentRound *int
	ParentRef   string
	Round       int // provisional; the authoritative number is taken at export
	CreatedAt   time.Time
}

// Server holds the snapshot cache and the packet store.
type Server struct {
	store  packet.Store
	assets fs.FS
	sample []byte
	opts   snapshot.Options

	mu    sync.RWMutex
	snaps map[string]*Snapshot
	seq   int
}

// New builds a Server. assets must contain web/app.html; sample is the
// embedded demo article (may be nil).
func New(store packet.Store, assets fs.FS, sample []byte, opts snapshot.Options) *Server {
	return &Server{
		store:  store,
		assets: assets,
		sample: sample,
		opts:   opts,
		snaps:  map[string]*Snapshot{},
	}
}

// Handler returns the mux. Routes match the tool's contract:
//
//	GET  /                 the app
//	GET  /view/{id}        the app, focused on a snapshot
//	GET  /js/{name}        a frontend script from web/
//	POST /snapshot         {url|path|sample} -> snapshot id + session/round
//	GET  /snapshot/{id}    the frozen HTML + metadata (for iframe srcdoc)
//	GET  /inbox            sessions/rounds listing
//
// There is no export route: a round is downloaded by the browser as a single
// self-contained bundle, so nothing leaves the page through this server.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleApp)
	mux.HandleFunc("GET /view/{id}", s.handleApp)
	mux.HandleFunc("GET /js/{name}", s.handleAsset)
	mux.HandleFunc("POST /snapshot", s.handleCreateSnapshot)
	mux.HandleFunc("GET /snapshot/{id}", s.handleGetSnapshot)
	mux.HandleFunc("GET /inbox", s.handleInbox)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "inbox": s.store.Root()})
	})
	return logging(mux)
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		if r.URL.Path != "/healthz" {
			fmt.Printf("  %-6s %-28s %s\n", r.Method, truncate(r.URL.Path, 28), time.Since(start).Round(time.Millisecond))
		}
	})
}

func (s *Server) handleApp(w http.ResponseWriter, r *http.Request) {
	b, err := fs.ReadFile(s.assets, "web/app.html")
	if err != nil {
		http.Error(w, "app.html missing from build", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(b)
}

// handleAsset serves a static frontend file from the asset FS. Only files
// directly under web/ are reachable, and the name is a single path segment,
// so it cannot traverse.
func (s *Server) handleAsset(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" || strings.ContainsAny(name, `/\.`) != strings.HasSuffix(name, ".js") {
		http.NotFound(w, r)
		return
	}
	b, err := fs.ReadFile(s.assets, "web/"+name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Write(b)
}

type snapshotRequest struct {
	URL    string `json:"url"`
	Path   string `json:"path"`
	Sample bool   `json:"sample"`
}

type snapshotResponse struct {
	ID          string        `json:"id"`
	ViewURL     string        `json:"view_url"`
	SessionID   string        `json:"session_id"`
	Round       int           `json:"round"`
	ParentRound *int          `json:"parent_round"`
	ParentRef   string        `json:"parent_ref,omitempty"`
	Source      packet.Source `json:"source"`
	Title       string        `json:"title"`
	Notes       []string      `json:"notes,omitempty"`
	Continues   bool          `json:"continues_session"`
}

func (s *Server) handleCreateSnapshot(w http.ResponseWriter, r *http.Request) {
	var req snapshotRequest
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	ctx := r.Context()

	var (
		res *snapshot.Result
		err error
		src packet.Source
	)

	switch {
	case req.Sample || strings.EqualFold(strings.TrimSpace(req.Path), "sample:article"):
		if s.sample == nil {
			writeErr(w, http.StatusNotFound, errors.New("no sample bundled"))
			return
		}
		res, err = snapshot.FromBytes(ctx, s.sample, nil, "sample", "sample/article.html", s.opts)
		src.Kind = "sample"
		src.Ref = "sample/article.html"
	case strings.TrimSpace(req.URL) != "":
		res, err = snapshot.FromURL(ctx, req.URL, s.opts)
		src.Kind = "url"
		src.Ref = strings.TrimSpace(req.URL)
	case strings.TrimSpace(req.Path) != "":
		p := expandPath(strings.TrimSpace(req.Path))
		// A file:// URL or a bare path both work.
		if u, uerr := url.Parse(p); uerr == nil && u.Scheme == "file" {
			p = filepath.FromSlash(u.Path)
		}
		res, err = snapshot.FromFile(ctx, p, s.opts)
		src.Kind = "file"
		if abs, aerr := filepath.Abs(p); aerr == nil {
			src.Ref = abs
		} else {
			src.Ref = p
		}
	default:
		writeErr(w, http.StatusBadRequest, errors.New("provide url, path, or sample"))
		return
	}
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}

	src.Title = res.Title
	src.FetchedAt = time.Now().UTC().Format(time.RFC3339)
	src.Notes = res.Notes

	snap := &Snapshot{
		ID:        s.nextID(),
		HTML:      res.HTML,
		Source:    src,
		CreatedAt: time.Now().UTC(),
	}

	// The loop: if this document is a previous round's output, the new packet
	// continues that session instead of starting a fresh one.
	continues := false
	if src.Kind == "file" || src.Kind == "result" {
		if session, round, name, ok := s.store.Locate(src.Ref); ok {
			snap.SessionID = session
			pr := round
			snap.ParentRound = &pr
			snap.ParentRef = fmt.Sprintf("round-%d/%s", round, name)
			if name == packet.FileResult {
				snap.Source.Kind = "result"
			}
			continues = true
		}
	}
	if snap.SessionID == "" {
		snap.SessionID = s.store.NewSessionID()
	}
	next, err := s.store.NextRound(snap.SessionID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	snap.Round = next

	s.mu.Lock()
	s.snaps[snap.ID] = snap
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, snapshotResponse{
		ID:          snap.ID,
		ViewURL:     "/view/" + snap.ID,
		SessionID:   snap.SessionID,
		Round:       snap.Round,
		ParentRound: snap.ParentRound,
		ParentRef:   snap.ParentRef,
		Source:      snap.Source,
		Title:       snap.Source.Title,
		Notes:       res.Notes,
		Continues:   continues,
	})
}

func (s *Server) handleGetSnapshot(w http.ResponseWriter, r *http.Request) {
	snap := s.lookup(r.PathValue("id"))
	if snap == nil {
		writeErr(w, http.StatusNotFound, errors.New("unknown snapshot"))
		return
	}
	// Round may have moved on since the snapshot was taken (another export).
	round, err := s.store.NextRound(snap.SessionID)
	if err == nil {
		snap.Round = round
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":                snap.ID,
		"html":              snap.HTML,
		"source":            snap.Source,
		"session_id":        snap.SessionID,
		"round":             snap.Round,
		"parent_round":      snap.ParentRound,
		"parent_ref":        snap.ParentRef,
		"continues_session": snap.ParentRound != nil,
		"inbox_root":        s.store.Root(),
	})
}

func (s *Server) handleInbox(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.store.Sessions()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if sessions == nil {
		sessions = []packet.SessionInfo{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"root":     s.store.Root(),
		"sessions": sessions,
	})
}

func (s *Server) lookup(id string) *Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snaps[id]
}

func (s *Server) nextID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	return fmt.Sprintf("snap-%d-%d", time.Now().Unix()%100000, s.seq)
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func readJSON(r *http.Request, dst any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxJSONBytes))
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return errors.New("empty body")
	}
	return json.Unmarshal(body, dst)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func writeErr(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]any{"error": err.Error()})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

// Shutdown is a convenience for the CLI.
func Shutdown(srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
