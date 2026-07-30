// Package snapshot turns a URL, a local file, or a previous round's
// result.html into one self-contained HTML document that can be annotated
// offline.
//
// Best-effort by design: static and server-rendered pages inline well, SPAs
// degrade. Scripts are always stripped -- the snapshot must not change under
// the author while they are marking it up, and the annotation anchors are
// only meaningful against a frozen DOM.
package snapshot

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Options tunes the best-effort inliner.
type Options struct {
	Client        *http.Client
	MaxAssetBytes int64
	MaxTotalBytes int64
	Timeout       time.Duration
	// MaxNotes caps how many best-effort warnings we keep.
	MaxNotes int
}

// DefaultOptions are sane values for a laptop on conference wifi.
func DefaultOptions() Options {
	return Options{
		Client:        &http.Client{Timeout: 12 * time.Second},
		MaxAssetBytes: 3 << 20,
		MaxTotalBytes: 24 << 20,
		Timeout:       25 * time.Second,
		MaxNotes:      25,
	}
}

func (o *Options) fill() {
	d := DefaultOptions()
	if o.Client == nil {
		o.Client = d.Client
	}
	if o.MaxAssetBytes <= 0 {
		o.MaxAssetBytes = d.MaxAssetBytes
	}
	if o.MaxTotalBytes <= 0 {
		o.MaxTotalBytes = d.MaxTotalBytes
	}
	if o.Timeout <= 0 {
		o.Timeout = d.Timeout
	}
	if o.MaxNotes <= 0 {
		o.MaxNotes = d.MaxNotes
	}
}

// Result is one frozen document plus what we could not inline.
type Result struct {
	HTML  string
	Title string
	Kind  string // "url" or "file"
	Ref   string
	Notes []string
}

// FromURL fetches an http(s) page and inlines it.
func FromURL(ctx context.Context, raw string, opt Options) (*Result, error) {
	opt.fill()
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
	ctx, cancel := context.WithTimeout(ctx, opt.Timeout)
	defer cancel()

	in := &inliner{opt: opt, ctx: ctx, base: u}
	body, _, err := in.get(u)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", u, err)
	}
	return in.run(body, "url", u.String())
}

// FromFile reads a local HTML file (including a previous round's result.html)
// and inlines assets relative to it.
func FromFile(ctx context.Context, path string, opt Options) (*Result, error) {
	opt.fill()
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if st.IsDir() {
		return nil, fmt.Errorf("%s is a directory", abs)
	}
	body, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, opt.Timeout)
	defer cancel()
	base := &url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}
	in := &inliner{opt: opt, ctx: ctx, base: base}
	return in.run(body, "file", abs)
}

// FromBytes inlines an in-memory document (used for the embedded sample).
func FromBytes(ctx context.Context, body []byte, base *url.URL, kind, ref string, opt Options) (*Result, error) {
	opt.fill()
	ctx, cancel := context.WithTimeout(ctx, opt.Timeout)
	defer cancel()
	in := &inliner{opt: opt, ctx: ctx, base: base}
	return in.run(body, kind, ref)
}

type inliner struct {
	opt   Options
	ctx   context.Context
	base  *url.URL
	used  int64
	notes []string
}

func (in *inliner) note(format string, args ...any) {
	if len(in.notes) >= in.opt.MaxNotes {
		return
	}
	in.notes = append(in.notes, fmt.Sprintf(format, args...))
}

// get resolves http(s) and file URLs uniformly.
func (in *inliner) get(u *url.URL) ([]byte, string, error) {
	if in.used >= in.opt.MaxTotalBytes {
		return nil, "", errors.New("asset budget exhausted")
	}
	switch u.Scheme {
	case "file", "":
		b, err := os.ReadFile(filepath.FromSlash(u.Path))
		if err != nil {
			return nil, "", err
		}
		if int64(len(b)) > in.opt.MaxAssetBytes {
			return nil, "", fmt.Errorf("asset too large (%d bytes)", len(b))
		}
		in.used += int64(len(b))
		return b, mime.TypeByExtension(strings.ToLower(filepath.Ext(u.Path))), nil
	case "http", "https":
		req, err := http.NewRequestWithContext(in.ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, "", err
		}
		req.Header.Set("User-Agent", "redline/1.0 (+local annotation tool)")
		req.Header.Set("Accept", "*/*")
		resp, err := in.opt.Client.Do(req)
		if err != nil {
			return nil, "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, "", fmt.Errorf("http %d", resp.StatusCode)
		}
		b, err := io.ReadAll(io.LimitReader(resp.Body, in.opt.MaxAssetBytes+1))
		if err != nil {
			return nil, "", err
		}
		if int64(len(b)) > in.opt.MaxAssetBytes {
			return nil, "", fmt.Errorf("asset too large (>%d bytes)", in.opt.MaxAssetBytes)
		}
		in.used += int64(len(b))
		return b, resp.Header.Get("Content-Type"), nil
	default:
		return nil, "", fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
}

func (in *inliner) resolve(ref string) (*url.URL, bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" || strings.HasPrefix(ref, "#") {
		return nil, false
	}
	if strings.HasPrefix(strings.ToLower(ref), "data:") ||
		strings.HasPrefix(strings.ToLower(ref), "javascript:") ||
		strings.HasPrefix(strings.ToLower(ref), "about:") {
		return nil, false
	}
	u, err := url.Parse(ref)
	if err != nil {
		return nil, false
	}
	if in.base == nil {
		if u.IsAbs() {
			return u, true
		}
		return nil, false
	}
	return in.base.ResolveReference(u), true
}

// stripped elements never survive into a snapshot.
var stripTags = map[atom.Atom]bool{
	atom.Script:   true,
	atom.Noscript: true,
	atom.Iframe:   true,
	atom.Object:   true,
	atom.Embed:    true,
	atom.Applet:   true,
}

func (in *inliner) run(body []byte, kind, ref string) (*Result, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}
	in.walk(doc)
	in.ensureCharset(doc)

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return nil, fmt.Errorf("render html: %w", err)
	}
	return &Result{
		HTML:  buf.String(),
		Title: findTitle(doc),
		Kind:  kind,
		Ref:   ref,
		Notes: in.notes,
	}, nil
}

// isRedlineState reports whether node is redline's own inert state block.
//
// Scripts are stripped so that a snapshot cannot move under the author while
// they annotate it. A <script type="application/redline+json"> is never
// executed by any browser -- an unknown script type is inert data -- so
// keeping it cannot violate that invariant. Stripping it, on the other hand,
// destroys the round trip: a bundle reopened in redline arrives with every
// annotation it carries silently removed.
func isRedlineState(node *html.Node) bool {
	if node.DataAtom != atom.Script {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(attr(node, "type")), "application/redline+json")
}

func (in *inliner) walk(n *html.Node) {
	var remove []*html.Node
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode {
			if stripTags[node.DataAtom] && !isRedlineState(node) {
				remove = append(remove, node)
				return
			}
			in.rewriteElement(node)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(n)
	for _, r := range remove {
		if r.Parent != nil {
			r.Parent.RemoveChild(r)
		}
	}
}

func (in *inliner) rewriteElement(node *html.Node) {
	// Drop inline event handlers regardless of tag.
	filtered := node.Attr[:0]
	for _, a := range node.Attr {
		lk := strings.ToLower(a.Key)
		if strings.HasPrefix(lk, "on") {
			continue
		}
		if lk == "integrity" || lk == "nonce" {
			continue
		}
		filtered = append(filtered, a)
	}
	node.Attr = filtered

	switch node.DataAtom {
	case atom.Link:
		in.rewriteLink(node)
	case atom.Img:
		in.rewriteImg(node)
	case atom.Source:
		// <source> inside <picture>/<video> would win over the inlined
		// <img src>; drop the candidate lists instead.
		delAttr(node, "srcset")
		delAttr(node, "data-srcset")
	case atom.Style:
		if txt := node.FirstChild; txt != nil && txt.Type == html.TextNode {
			txt.Data = sanitizeCSS(in.rewriteCSS(txt.Data, in.base, 0))
		}
	case atom.A, atom.Area:
		in.absolutize(node, "href")
	case atom.Form:
		in.absolutize(node, "action")
	case atom.Base:
		// A <base> would re-point everything we just resolved. Neutralize it.
		delAttr(node, "href")
	}
}

func (in *inliner) rewriteLink(node *html.Node) {
	rel := strings.ToLower(strings.TrimSpace(attr(node, "rel")))
	href := attr(node, "href")
	as := strings.ToLower(attr(node, "as"))

	if rel == "preload" && (as == "script" || as == "fetch") || rel == "modulepreload" || rel == "prefetch" {
		delAttr(node, "href")
		return
	}
	if !strings.Contains(rel, "stylesheet") {
		if rel == "icon" || strings.Contains(rel, "icon") {
			in.absolutize(node, "href")
		}
		return
	}
	u, ok := in.resolve(href)
	if !ok {
		return
	}
	css, _, err := in.get(u)
	if err != nil {
		in.note("stylesheet not inlined (%s): %v", shorten(u.String()), err)
		in.absolutize(node, "href")
		return
	}
	// Turn <link rel=stylesheet> into <style>.
	node.Type = html.ElementNode
	node.DataAtom = atom.Style
	node.Data = "style"
	media := attr(node, "media")
	node.Attr = nil
	if media != "" && media != "all" {
		node.Attr = append(node.Attr, html.Attribute{Key: "media", Val: media})
	}
	for c := node.FirstChild; c != nil; {
		next := c.NextSibling
		node.RemoveChild(c)
		c = next
	}
	node.AppendChild(&html.Node{
		Type: html.TextNode,
		Data: sanitizeCSS(in.rewriteCSS(string(css), u, 0)),
	})
}

func (in *inliner) rewriteImg(node *html.Node) {
	src := attr(node, "src")
	if src == "" {
		// Common lazy-loading patterns.
		for _, k := range []string{"data-src", "data-original", "data-lazy-src"} {
			if v := attr(node, k); v != "" {
				src = v
				setAttr(node, "src", v)
				break
			}
		}
	}
	delAttr(node, "srcset")
	delAttr(node, "data-srcset")
	delAttr(node, "loading")
	if src == "" {
		return
	}
	if strings.HasPrefix(strings.ToLower(src), "data:") {
		return
	}
	u, ok := in.resolve(src)
	if !ok {
		return
	}
	data, ctype, err := in.get(u)
	if err != nil {
		in.note("image not inlined (%s): %v", shorten(u.String()), err)
		setAttr(node, "src", u.String())
		return
	}
	setAttr(node, "src", dataURI(data, ctype, u.Path))
}

var (
	// RE2 has no backreferences, so each quoting style gets its own group.
	cssURLRe    = regexp.MustCompile(`url\(\s*(?:'([^']*)'|"([^"]*)"|([^'")\s]*))\s*\)`)
	cssImportRe = regexp.MustCompile(`@import\s+(?:url\(\s*['"]?([^'")]+)['"]?\s*\)|['"]([^'"]+)['"])\s*;?`)
)

// rewriteCSS resolves relative url() references and pulls in one or two
// levels of @import so that most sites' stylesheets survive going offline.
func (in *inliner) rewriteCSS(css string, base *url.URL, depth int) string {
	if depth < 2 {
		css = cssImportRe.ReplaceAllStringFunc(css, func(m string) string {
			sub := cssImportRe.FindStringSubmatch(m)
			ref := sub[1]
			if ref == "" {
				ref = sub[2]
			}
			target, err := url.Parse(strings.TrimSpace(ref))
			if err != nil {
				return ""
			}
			abs := target
			if base != nil {
				abs = base.ResolveReference(target)
			}
			b, _, err := in.get(abs)
			if err != nil {
				in.note("@import not inlined (%s): %v", shorten(abs.String()), err)
				return ""
			}
			return in.rewriteCSS(string(b), abs, depth+1)
		})
	}
	if base == nil {
		return css
	}
	return cssURLRe.ReplaceAllStringFunc(css, func(m string) string {
		sub := cssURLRe.FindStringSubmatch(m)
		ref := sub[1]
		if ref == "" {
			ref = sub[2]
		}
		if ref == "" {
			ref = sub[3]
		}
		ref = strings.TrimSpace(ref)
		lower := strings.ToLower(ref)
		if ref == "" || strings.HasPrefix(lower, "data:") || strings.HasPrefix(ref, "#") {
			return m
		}
		target, err := url.Parse(ref)
		if err != nil {
			return m
		}
		abs := base.ResolveReference(target)
		// Small assets (icons, fonts) get inlined; anything bigger keeps an
		// absolute URL so an online run still looks right.
		if b, ctype, err := in.get(abs); err == nil && int64(len(b)) <= 512<<10 {
			return "url(\"" + dataURI(b, ctype, abs.Path) + "\")"
		}
		return "url(\"" + abs.String() + "\")"
	})
}

func (in *inliner) absolutize(node *html.Node, key string) {
	v := attr(node, key)
	if v == "" {
		return
	}
	if u, ok := in.resolve(v); ok {
		setAttr(node, key, u.String())
	}
}

func (in *inliner) ensureCharset(doc *html.Node) {
	head := findFirst(doc, atom.Head)
	if head == nil {
		return
	}
	for c := head.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.DataAtom == atom.Meta && attr(c, "charset") != "" {
			return
		}
	}
	meta := &html.Node{Type: html.ElementNode, DataAtom: atom.Meta, Data: "meta",
		Attr: []html.Attribute{{Key: "charset", Val: "utf-8"}}}
	if head.FirstChild != nil {
		head.InsertBefore(meta, head.FirstChild)
	} else {
		head.AppendChild(meta)
	}
}

func dataURI(data []byte, ctype, path string) string {
	ct := strings.TrimSpace(strings.Split(ctype, ";")[0])
	if ct == "" || ct == "application/octet-stream" {
		ct = mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
	}
	if ct == "" {
		ct = http.DetectContentType(data)
	}
	ct = strings.TrimSpace(strings.Split(ct, ";")[0])
	if ct == "" {
		ct = "application/octet-stream"
	}
	return "data:" + ct + ";base64," + base64.StdEncoding.EncodeToString(data)
}

// sanitizeCSS makes sure inlined CSS cannot terminate its own <style> block.
func sanitizeCSS(css string) string {
	return strings.ReplaceAll(css, "</style", "<\\/style")
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

func setAttr(n *html.Node, key, val string) {
	for i, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			n.Attr[i].Val = val
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: key, Val: val})
}

func delAttr(n *html.Node, key string) {
	out := n.Attr[:0]
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			continue
		}
		out = append(out, a)
	}
	n.Attr = out
}

func findFirst(n *html.Node, a atom.Atom) *html.Node {
	if n.Type == html.ElementNode && n.DataAtom == a {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if got := findFirst(c, a); got != nil {
			return got
		}
	}
	return nil
}

func findTitle(doc *html.Node) string {
	t := findFirst(doc, atom.Title)
	if t == nil || t.FirstChild == nil {
		if h1 := findFirst(doc, atom.H1); h1 != nil {
			return strings.TrimSpace(textOf(h1))
		}
		return ""
	}
	return strings.TrimSpace(textOf(t))
}

func textOf(n *html.Node) string {
	var b strings.Builder
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(n)
	return b.String()
}

func shorten(s string) string {
	if len(s) <= 70 {
		return s
	}
	return s[:40] + "..." + s[len(s)-25:]
}
