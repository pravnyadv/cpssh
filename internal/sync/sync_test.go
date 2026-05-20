package sync

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pravnyadv/cpssh/internal/config"
)

func TestBuildRemoteCmd_NoPruneByDefault(t *testing.T) {
	got := buildRemoteCmd("$HOME/.cpssh", "img3.png", 10, false)
	want := `cat > "$HOME/.cpssh/img3.png"`
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestBuildRemoteCmd_PrunesWhenFlagged(t *testing.T) {
	got := buildRemoteCmd("$HOME/.cpssh", "img10.png", 10, true)
	for _, want := range []string{
		`cat > "$HOME/.cpssh/img10.png"`,
		`cd "$HOME/.cpssh"`,
		`tail -n +11`,
		`xargs -r rm -f`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected substring %q in command:\n  %s", want, got)
		}
	}
}

func TestBuildRemoteCmd_UsesXargsDashR(t *testing.T) {
	// Regression guard: GNU xargs runs the utility even on empty input, so
	// plain `xargs rm -f` fails on fresh Linux servers with <keepN files.
	// `-r` skips empty invocations on GNU and is a no-op on BSD/macOS.
	got := buildRemoteCmd("/tmp/x", "img10.png", 10, true)
	if !strings.Contains(got, "xargs -r rm -f") {
		t.Errorf("missing `xargs -r rm -f` in prune command:\n  %s", got)
	}
}

func TestBuildRemoteCmd_PruneOrderIsCorrect(t *testing.T) {
	got := buildRemoteCmd("/p", "img10.png", 5, true)
	order := []string{"cat >", "cd ", "tail -n +6"}
	prev := -1
	for _, s := range order {
		idx := strings.Index(got, s)
		if idx < 0 {
			t.Fatalf("missing %q in: %s", s, got)
		}
		if idx <= prev {
			t.Errorf("step %q appeared before previous step in: %s", s, got)
		}
		prev = idx
	}
}

func TestMaybeCompressBytes_ReturnsOriginalWhenSmall(t *testing.T) {
	cfg := &config.Config{Settings: config.Settings{CompressAboveKB: 500}}
	in := bytes.Repeat([]byte{0x89}, 1024) // 1 KB

	out := maybeCompressBytes(cfg, in)
	if &out[0] != &in[0] {
		t.Errorf("expected same underlying slice when no compression triggered")
	}
}

func TestMaybeCompressBytes_FallsBackOnToolFailure(t *testing.T) {
	cfg := &config.Config{Settings: config.Settings{CompressAboveKB: 1}}
	in := bytes.Repeat([]byte{0x00}, 2048) // 2 KB > 1 KB threshold

	out := maybeCompressBytes(cfg, in)
	if !bytes.Equal(out, in) {
		t.Errorf("expected fallback to original data when compression fails")
	}
}
