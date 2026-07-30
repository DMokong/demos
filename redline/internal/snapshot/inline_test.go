package snapshot

import (
	"context"
	"strings"
	"testing"
)

// The snapshot pipeline strips scripts so a page cannot move under the author
// while they annotate it. redline's own state block is a <script> too, but with
// an unknown type, so no browser executes it -- it is inert data carrying the
// annotations. Stripping it silently empties every bundle reopened in redline,
// which is the whole return path of the loop. These two tests pin both halves:
// the block survives, and real scripts still do not.
func TestRedlineStateBlockSurvivesInlining(t *testing.T) {
	const src = `<!doctype html><html><body>
<p>Body text.</p>
<script type="application/redline+json" id="redline-state">
{"schema_version":"2.0","document_key":"sample:sample/article.html","round":1}
</script>
</body></html>`

	res, err := FromBytes(context.Background(), []byte(src), nil, "file", "t.html", DefaultOptions())
	if err != nil {
		t.Fatalf("FromBytes: %v", err)
	}
	got := string(res.HTML)

	if !strings.Contains(got, `id="redline-state"`) {
		t.Errorf("state block was stripped from the snapshot; bundles would reopen with no annotations\n%s", got)
	}
	if !strings.Contains(got, `"document_key":"sample:sample/article.html"`) {
		t.Errorf("state block survived but its payload did not\n%s", got)
	}
}

func TestExecutableScriptsAreStillStripped(t *testing.T) {
	const src = `<!doctype html><html><body>
<p>Body text.</p>
<script>window.x = 1;</script>
<script type="text/javascript">window.y = 2;</script>
<script type="module">export const z = 3;</script>
</body></html>`

	res, err := FromBytes(context.Background(), []byte(src), nil, "file", "t.html", DefaultOptions())
	if err != nil {
		t.Fatalf("FromBytes: %v", err)
	}
	got := string(res.HTML)

	for _, banned := range []string{"window.x", "window.y", "export const z", "<script>"} {
		if strings.Contains(got, banned) {
			t.Errorf("executable script survived inlining: found %q\n%s", banned, got)
		}
	}
}
