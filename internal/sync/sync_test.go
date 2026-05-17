package sync

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pravnyadv/cpssh/internal/config"
)

func TestBuildRemoteCmd_PrunesByKeepN(t *testing.T) {
	got := buildRemoteCmd("$HOME/.cpssh", "img3.png", 10)

	// Each path interpolation must be wrapped in double quotes so a path with
	// spaces would still parse on the remote shell.
	for _, want := range []string{
		`mkdir -p "$HOME/.cpssh"`,
		`cat > "$HOME/.cpssh/img3.png"`,
		`cd "$HOME/.cpssh"`,
		`ln -sf "img3.png" latest.png`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected substring %q in command:\n  %s", want, got)
		}
	}

	// keep_last_n=10 → skip the first 10 lines of `ls -t`, prune the rest.
	if !strings.Contains(got, "tail -n +11") {
		t.Errorf("expected tail -n +11 (keepN+1=11), got:\n  %s", got)
	}
}

func TestBuildRemoteCmd_UsesXargsDashR(t *testing.T) {
	got := buildRemoteCmd("/tmp/x", "img1.png", 10)
	// Regression guard for the original bug: GNU xargs runs the utility even
	// on empty input, so plain `xargs rm -f` fails (rm with no operands) on
	// every fresh Linux server. `-r` skips empty invocations on GNU and is a
	// documented no-op on BSD/macOS.
	if !strings.Contains(got, "xargs -r rm -f") {
		t.Errorf("missing `xargs -r rm -f` — the chain would fail on Linux servers with <keepN files. cmd:\n  %s", got)
	}
}

func TestMaybeCompressBytes_ReturnsOriginalWhenSmall(t *testing.T) {
	// Below the threshold: must NOT touch the filesystem and must return the
	// same slice we passed in (identity check, not just byte-equality).
	cfg := &config.Config{Settings: config.Settings{CompressAboveKB: 500}}
	in := bytes.Repeat([]byte{0x89}, 1024) // 1 KB

	out := maybeCompressBytes(cfg, in)
	if &out[0] != &in[0] {
		t.Errorf("expected same underlying slice when no compression triggered")
	}
}

func TestMaybeCompressBytes_FallsBackOnToolFailure(t *testing.T) {
	// Above the threshold but the compression tool (sips/convert) won't accept
	// our garbage bytes — should fall back to returning the original data
	// rather than failing the sync entirely.
	cfg := &config.Config{Settings: config.Settings{CompressAboveKB: 1}}
	in := bytes.Repeat([]byte{0x00}, 2048) // 2 KB > 1 KB threshold

	out := maybeCompressBytes(cfg, in)
	if !bytes.Equal(out, in) {
		t.Errorf("expected fallback to original data when compression fails")
	}
}

func TestBuildRemoteCmd_ChainOrderIsCorrect(t *testing.T) {
	// The full chain must do: mkdir → cat → cd → ln → prune. Re-ordering
	// (e.g. pruning before writing) would break the keep-N invariant.
	got := buildRemoteCmd("/p", "img2.png", 5)
	order := []string{"mkdir -p", "cat >", "cd ", "ln -sf", "tail -n +6"}
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
