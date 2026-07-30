// Package redline embeds the frontend and the bundled demo target so that
// `go build ./cmd/redline` produces one self-contained binary.
//
// The embed directive has to live in a package whose directory contains the
// files, which is why this one file sits at the module root.
package redline

import "embed"

// Assets holds the frontend (web/app.html plus its scripts) and
// sample/article.html (the bundled demo draft).
//
//go:embed web/app.html web/redline-core.js web/redline-store.js sample/article.html
var Assets embed.FS
