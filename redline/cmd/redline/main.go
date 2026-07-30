// Command redline serves the annotation UI. Rounds are downloaded by the
// browser as self-contained bundles; nothing is written here.
//
//	go build ./cmd/redline && ./redline serve
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"redline"
	"redline/internal/server"
	"redline/internal/snapshot"
)

const usage = `redline - annotate a page, download a round, hand it to an agent.

usage:
  redline serve [-port 8787] [-addr 127.0.0.1]

flags:
  -port    port to listen on (default 8787)
  -addr    interface to bind (default 127.0.0.1; use 0.0.0.0 to share)
`

func main() {
	if len(os.Args) < 2 || os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help" {
		fmt.Print(usage)
		if len(os.Args) < 2 {
			os.Exit(2)
		}
		return
	}
	switch os.Args[1] {
	case "serve":
		if err := serve(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "redline:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "redline: unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
}

func serve(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 8787, "port to listen on")
	addr := fs.String("addr", "127.0.0.1", "interface to bind")
	if err := fs.Parse(args); err != nil {
		return err
	}

	sample, err := readSample()
	if err != nil {
		return err
	}

	srv := server.New(assetFS(), sample, snapshot.DefaultOptions())
	hostport := net.JoinHostPort(*addr, fmt.Sprint(*port))
	httpSrv := &http.Server{
		Addr:              hostport,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		// A snapshot inlines every image as a data URI; give it room.
		WriteTimeout: 120 * time.Second,
		ReadTimeout:  120 * time.Second,
	}

	ln, err := net.Listen("tcp", hostport)
	if err != nil {
		return err
	}

	fmt.Printf(`
  redline

  ui      http://%s/

  open the sample article from the UI, annotate it, hit Export.
  then: "Process the redline bundle at <the file you just downloaded>."

`, hostport)

	idle := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		fmt.Println("\n  shutting down")
		server.Shutdown(httpSrv)
		close(idle)
	}()

	if err := httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
		return err
	}
	<-idle
	return nil
}

// assetFS prefers files on disk when they exist (so the frontend can be
// edited without rebuilding), otherwise the embedded copy.
func assetFS() fs.FS {
	if st, err := os.Stat(filepath.Join("web", "app.html")); err == nil && !st.IsDir() {
		return os.DirFS(".")
	}
	return redline.Assets
}

func readSample() ([]byte, error) {
	if b, err := os.ReadFile(filepath.Join("sample", "article.html")); err == nil {
		return b, nil
	}
	b, err := redline.Assets.ReadFile("sample/article.html")
	if err != nil {
		return nil, fmt.Errorf("sample article missing: %w", err)
	}
	return b, nil
}
