// Package server wires redline's HTTP surface.
//
// Every handler here is a plain http.Handler registered on a stdlib mux, and
// the server keeps no durable state at all: it freezes a page and serves the
// app. The round itself lives in the browser (IndexedDB) and travels as a
// downloaded bundle, so these handlers can mount inside the existing Go
// service later without dragging storage along.
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

	"redline/internal/snapshot"
)

const maxJSONBytes = 2 << 20

// Source records where the annotated content came from. The frontend derives
// a document identity from it, so the same page opened by path and by URL does
// not split into two annotation sets.
type Source struct {
	// Kind is "url", "file" or "sample".
	Kind      string   `json:"kind"`
	Ref       string   `json:"ref"`
	Title     string   `json:"title,omitempty"`
	FetchedAt string   `json:"fetched_at,omitempty"`
	Notes     []string `json:"notes,omitempty"`
}

// Snapshot is one frozen document held in memory for the life of the process.
type Snapshot struct {
	ID        string
	HTML      string
	Source    Source
	CreatedAt time.Time
}

// Server holds the snapshot cache.
type Server struct {
	assets fs.FS
	sample []byte
	opts   snapshot.Options

	mu    sync.RWMutex
	snaps map[string]*Snapshot
	seq   int
}

// New builds a Server. assets must contain web/app.html; sample is the
// embedded demo article (may be nil).
func New(assets fs.FS, sample []byte, opts snapshot.Options) *Server {
	return &Server{
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
//	POST /snapshot         {url|path|sample} -> snapshot id
//	GET  /snapshot/{id}    the frozen HTML + metadata (for iframe srcdoc)
//
// There is no export route and no round listing: a round is downloaded by the
// browser as one self-contained bundle, so nothing leaves through this server.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleApp)
	mux.HandleFunc("GET /view/{id}", s.handleApp)
	mux.HandleFunc("GET /js/{name}", s.handleAsset)
	mux.HandleFunc("POST /snapshot", s.handleCreateSnapshot)
	mux.HandleFunc("GET /snapshot/{id}", s.handleGetSnapshot)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
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
	ID      string   `json:"id"`
	ViewURL string   `json:"view_url"`
	Source  Source   `json:"source"`
	Title   string   `json:"title"`
	Notes   []string `json:"notes,omitempty"`
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
		src Source
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

	// No session bookkeeping lives here: the round number and the annotation
	// set belong to the document's record in the browser, keyed by source.
	snap := &Snapshot{
		ID:        s.nextID(),
		HTML:      res.HTML,
		Source:    src,
		CreatedAt: time.Now().UTC(),
	}

	s.mu.Lock()
	s.snaps[snap.ID] = snap
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, snapshotResponse{
		ID:      snap.ID,
		ViewURL: "/view/" + snap.ID,
		Source:  snap.Source,
		Title:   snap.Source.Title,
		Notes:   res.Notes,
	})
}

func (s *Server) handleGetSnapshot(w http.ResponseWriter, r *http.Request) {
	snap := s.lookup(r.PathValue("id"))
	if snap == nil {
		writeErr(w, http.StatusNotFound, errors.New("unknown snapshot"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":     snap.ID,
		"html":   snap.HTML,
		"source": snap.Source,
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
