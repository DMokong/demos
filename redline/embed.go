// Package redline embeds the frontend and the bundled demo target so that
// `go build ./cmd/redline` produces one self-contained binary.
//
// The embed directive has to live in a package whose directory contains the
// files, which is why this one file sits at the module root.
package redline

import "embed"

// Assets holds web/app.html (the entire frontend) and sample/article.html
// (the bundled demo draft).
//
//go:embed web/app.html sample/article.html
var Assets embed.FS
