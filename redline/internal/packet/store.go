// Package packet defines the on-disk shape of a redline round and the
// PacketStore seam that keeps it swappable.
//
// A "packet" is one turn in the human/agent co-authoring loop: everything the
// agent needs to continue the work, written into inbox/<session>/round-<N>/.
// LocalDir is the implementation used by the standalone demo binary; the
// interface exists so an S3-backed store can drop in when these handlers are
// mounted inside the Lambda service. Nothing else in the tree touches the
// filesystem layout directly.
package packet

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// SchemaVersion is stamped into manifest.json and annotations.json. The
// redline skill reads it before anything else.
const SchemaVersion = "1.0"

// Canonical file names inside a round directory.
const (
	FileManifest    = "manifest.json"
	FileSnapshot    = "snapshot.html"
	FileAnnotations = "annotations.json"
	FileAnnotated   = "annotated.png"
	FileResult      = "result.html"
	FileChanges     = "changes.md"
)

// Source records where the annotated content came from.
type Source struct {
	// Kind is "url", "file", "sample" or "result" (a previous round's output).
	Kind      string   `json:"kind"`
	Ref       string   `json:"ref"`
	Title     string   `json:"title,omitempty"`
	FetchedAt string   `json:"fetched_at,omitempty"`
	Notes     []string `json:"notes,omitempty"`
}

// Viewport is the geometry the annotations were captured at. The agent
// verifies its result at Width and at 390px.
type Viewport struct {
	Width            int     `json:"width"`
	Height           int     `json:"height"`
	GutterWidth      int     `json:"gutter_width"`
	DevicePixelRatio float64 `json:"device_pixel_ratio"`
}

// Counts is a quick summary so a human (or an orchestrator deciding how to
// route the round) can size the work without parsing annotations.json.
type Counts struct {
	Draw       int            `json:"draw"`
	Highlights int            `json:"highlights"`
	ByIntent   map[string]int `json:"by_intent"`
	ByDrawType map[string]int `json:"by_draw_type"`
}

// Manifest is manifest.json.
type Manifest struct {
	SchemaVersion string   `json:"schema_version"`
	Tool          string   `json:"tool"`
	SessionID     string   `json:"session_id"`
	Round         int      `json:"round"`
	ParentRound   *int     `json:"parent_round"`
	ParentRef     string   `json:"parent_ref,omitempty"`
	CreatedAt     string   `json:"created_at"`
	Source        Source   `json:"source"`
	Viewport      Viewport `json:"viewport"`
	Counts        Counts   `json:"counts"`
	Files         []string `json:"files"`
	Notes         []string `json:"notes,omitempty"`
	NextStep      string   `json:"next_step"`
}

// File is one blob to write into a round directory.
type File struct {
	Name string
	Data []byte
}

// RoundInfo describes one round directory for the UI's inbox browser.
type RoundInfo struct {
	Round       int      `json:"round"`
	Dir         string   `json:"dir"`
	Files       []string `json:"files"`
	HasSnapshot bool     `json:"has_snapshot"`
	HasResult   bool     `json:"has_result"`
	HasChanges  bool     `json:"has_changes"`
	// Pending means: a packet is here but the agent has not written result.html.
	Pending   bool    `json:"pending"`
	CreatedAt string  `json:"created_at"`
	Source    *Source `json:"source,omitempty"`
	Counts    *Counts `json:"counts,omitempty"`
}

// SessionInfo groups the rounds of one co-authoring session.
type SessionInfo struct {
	SessionID string      `json:"session_id"`
	Title     string      `json:"title,omitempty"`
	Rounds    []RoundInfo `json:"rounds"`
	UpdatedAt string      `json:"updated_at"`
}

// Store is the seam: LocalDir today, S3 later. Handlers depend on this, never
// on the filesystem.
type Store interface {
	// Root is the human-readable base location (e.g. "inbox").
	Root() string
	// NewSessionID mints an id for a session that does not exist yet.
	NewSessionID() string
	// NextRound returns the round number a new packet for this session gets.
	NextRound(session string) (int, error)
	// WriteRound writes files into <session>/round-<n>/ and returns a
	// display path for that directory.
	WriteRound(session string, round int, files []File) (string, error)
	// Sessions lists everything for the inbox browser, newest session last.
	Sessions() ([]SessionInfo, error)
	// ReadRoundFile reads one file out of a round directory.
	ReadRoundFile(session string, round int, name string) ([]byte, error)
	// Locate maps an arbitrary path onto (session, round, file) when it
	// points inside this store. Used to detect "the user opened a previous
	// round's result.html" so the packet continues that session.
	Locate(path string) (session string, round int, name string, ok bool)
}

var roundDirRe = regexp.MustCompile(`^round-(\d+)$`)

// LocalDir stores packets under a directory (./inbox by default).
type LocalDir struct {
	root string
}

// NewLocalDir creates the root directory if needed.
func NewLocalDir(root string) (*LocalDir, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}
	return &LocalDir{root: abs}, nil
}

var _ Store = (*LocalDir)(nil)

// Root returns a short display path for the inbox root.
func (l *LocalDir) Root() string { return display(l.root) }

// AbsRoot returns the absolute inbox root.
func (l *LocalDir) AbsRoot() string { return l.root }

// NewSessionID mints a readable, sortable session id.
func (l *LocalDir) NewSessionID() string {
	var b [3]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Time-based fallback; uniqueness matters, secrecy does not.
		return fmt.Sprintf("s-%s-%06d", time.Now().Format("20060102"), time.Now().Nanosecond()%1000000)
	}
	return fmt.Sprintf("s-%s-%s", time.Now().Format("20060102"), hex.EncodeToString(b[:]))
}

func (l *LocalDir) sessionDir(session string) (string, error) {
	if !validSegment(session) {
		return "", fmt.Errorf("invalid session id %q", session)
	}
	return filepath.Join(l.root, session), nil
}

// NextRound returns max(existing round)+1, or 1 for an unknown session.
func (l *LocalDir) NextRound(session string) (int, error) {
	dir, err := l.sessionDir(session)
	if err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return 1, nil
	}
	if err != nil {
		return 0, err
	}
	max := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if m := roundDirRe.FindStringSubmatch(e.Name()); m != nil {
			n, _ := strconv.Atoi(m[1])
			if n > max {
				max = n
			}
		}
	}
	return max + 1, nil
}

// WriteRound writes the packet files and returns the round directory path.
func (l *LocalDir) WriteRound(session string, round int, files []File) (string, error) {
	dir, err := l.sessionDir(session)
	if err != nil {
		return "", err
	}
	if round < 1 {
		return "", fmt.Errorf("invalid round %d", round)
	}
	rd := filepath.Join(dir, fmt.Sprintf("round-%d", round))
	if err := os.MkdirAll(rd, 0o755); err != nil {
		return "", err
	}
	for _, f := range files {
		if f.Name == "" || strings.ContainsAny(f.Name, `/\`) {
			return "", fmt.Errorf("invalid file name %q", f.Name)
		}
		if err := os.WriteFile(filepath.Join(rd, f.Name), f.Data, 0o644); err != nil {
			return "", err
		}
	}
	return display(rd), nil
}

// ReadRoundFile reads one file from a round directory.
func (l *LocalDir) ReadRoundFile(session string, round int, name string) ([]byte, error) {
	dir, err := l.sessionDir(session)
	if err != nil {
		return nil, err
	}
	if strings.ContainsAny(name, `/\`) {
		return nil, fmt.Errorf("invalid file name %q", name)
	}
	return os.ReadFile(filepath.Join(dir, fmt.Sprintf("round-%d", round), name))
}

// Sessions walks the inbox for the UI's "open a previous result" browser.
func (l *LocalDir) Sessions() ([]SessionInfo, error) {
	entries, err := os.ReadDir(l.root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []SessionInfo
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		si := SessionInfo{SessionID: e.Name()}
		rounds, err := os.ReadDir(filepath.Join(l.root, e.Name()))
		if err != nil {
			continue
		}
		for _, r := range rounds {
			if !r.IsDir() {
				continue
			}
			m := roundDirRe.FindStringSubmatch(r.Name())
			if m == nil {
				continue
			}
			n, _ := strconv.Atoi(m[1])
			rd := filepath.Join(l.root, e.Name(), r.Name())
			ri := RoundInfo{Round: n, Dir: display(rd)}
			fs, err := os.ReadDir(rd)
			if err != nil {
				continue
			}
			for _, f := range fs {
				if f.IsDir() {
					continue
				}
				ri.Files = append(ri.Files, f.Name())
				switch f.Name() {
				case FileSnapshot:
					ri.HasSnapshot = true
				case FileResult:
					ri.HasResult = true
				case FileChanges:
					ri.HasChanges = true
				}
			}
			sort.Strings(ri.Files)
			ri.Pending = ri.HasSnapshot && !ri.HasResult
			if info, err := os.Stat(rd); err == nil {
				ri.CreatedAt = info.ModTime().UTC().Format(time.RFC3339)
				if ri.CreatedAt > si.UpdatedAt {
					si.UpdatedAt = ri.CreatedAt
				}
			}
			if b, err := os.ReadFile(filepath.Join(rd, FileManifest)); err == nil {
				var mf Manifest
				if json.Unmarshal(b, &mf) == nil {
					src := mf.Source
					ri.Source = &src
					c := mf.Counts
					ri.Counts = &c
					if mf.CreatedAt != "" {
						ri.CreatedAt = mf.CreatedAt
					}
					if si.Title == "" && mf.Source.Title != "" {
						si.Title = mf.Source.Title
					}
				}
			}
			si.Rounds = append(si.Rounds, ri)
		}
		sort.Slice(si.Rounds, func(i, j int) bool { return si.Rounds[i].Round < si.Rounds[j].Round })
		if len(si.Rounds) == 0 {
			continue
		}
		out = append(out, si)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt != out[j].UpdatedAt {
			return out[i].UpdatedAt > out[j].UpdatedAt
		}
		return out[i].SessionID < out[j].SessionID
	})
	return out, nil
}

// Locate reports whether path points at a file inside <root>/<session>/round-N/.
// This is how "open a previous round's result.html" continues a session
// instead of starting a new one.
func (l *LocalDir) Locate(path string) (string, int, string, bool) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", 0, "", false
	}
	rel, err := filepath.Rel(l.root, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", 0, "", false
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) != 3 {
		return "", 0, "", false
	}
	m := roundDirRe.FindStringSubmatch(parts[1])
	if m == nil {
		return "", 0, "", false
	}
	n, _ := strconv.Atoi(m[1])
	return parts[0], n, parts[2], true
}

func validSegment(s string) bool {
	if s == "" || s == "." || s == ".." {
		return false
	}
	if strings.ContainsAny(s, `/\`) {
		return false
	}
	return true
}

// display shortens an absolute path to a CWD-relative one when that is
// shorter, so the UI can show "inbox/s-.../round-1".
func display(p string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return p
	}
	rel, err := filepath.Rel(cwd, p)
	if err != nil || strings.HasPrefix(rel, "..") {
		return p
	}
	return filepath.ToSlash(rel)
}
