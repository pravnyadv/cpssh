package cmd

import (
	"bufio"
	"strings"
	"testing"

	"github.com/pravnyadv/cpssh/internal/config"
)

func TestParseServerAddress(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		defaultUser string
		wantUser    string
		wantHost    string
		wantPort    int
		wantErr     string
	}{
		{name: "user@host", raw: "alice@example.com", wantUser: "alice", wantHost: "example.com"},
		{name: "user@host:port", raw: "alice@example.com:2222", wantUser: "alice", wantHost: "example.com", wantPort: 2222},
		{name: "host only uses default user", raw: "example.com", defaultUser: "bob", wantUser: "bob", wantHost: "example.com"},
		{name: "host:port only", raw: "example.com:443", defaultUser: "bob", wantUser: "bob", wantHost: "example.com", wantPort: 443},
		{name: "user with @ in it splits at first @", raw: "weird@name@host", wantUser: "weird", wantHost: "name@host"},

		{name: "empty rejected", raw: "", wantErr: "empty"},
		{name: "no user and no default rejected", raw: "example.com", wantErr: "user cannot be empty"},
		{name: "missing host rejected", raw: "alice@", wantErr: "host cannot be empty"},
		{name: "non-numeric port rejected", raw: "alice@example.com:abc", wantErr: "invalid port"},
		{name: "negative port rejected", raw: "alice@example.com:-1", wantErr: "invalid port"},
		{name: "port out of range rejected", raw: "alice@example.com:70000", wantErr: "invalid port"},
		{name: "port zero rejected", raw: "alice@example.com:0", wantErr: "invalid port"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, host, port, err := parseServerAddress(tt.raw, tt.defaultUser)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil (user=%q host=%q port=%d)", tt.wantErr, user, host, port)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error %q does not contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if user != tt.wantUser || host != tt.wantHost || port != tt.wantPort {
				t.Errorf("got (user=%q host=%q port=%d), want (user=%q host=%q port=%d)",
					user, host, port, tt.wantUser, tt.wantHost, tt.wantPort)
			}
		})
	}
}

func TestNormalizeSyncPath(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", "$HOME/.cpssh"},
		{"~", "$HOME"},
		{"~/foo", "$HOME/foo"},
		{"~/.cpssh", "$HOME/.cpssh"},
		{"$HOME/.cpssh", "$HOME/.cpssh"},                       // already shell form, unchanged
		{"/var/data/cpssh", "/var/data/cpssh"},                 // absolute kept as-is
		{"~user/foo", "~user/foo"},                             // only ~/ prefix is rewritten, not ~user
		{"relative/path", "relative/path"},                     // not modified
	}
	for _, tt := range tests {
		if got := normalizeSyncPath(tt.in); got != tt.want {
			t.Errorf("normalizeSyncPath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestServerAddr_OmitsDefaultPort(t *testing.T) {
	tests := []struct {
		s    config.Server
		want string
	}{
		{config.Server{User: "alice", Host: "ex.com"}, "alice@ex.com"},
		{config.Server{User: "alice", Host: "ex.com", Port: 22}, "alice@ex.com"},
		{config.Server{User: "alice", Host: "ex.com", Port: 2222}, "alice@ex.com:2222"},
	}
	for _, tt := range tests {
		if got := serverAddr(tt.s); got != tt.want {
			t.Errorf("serverAddr(%+v) = %q, want %q", tt.s, got, tt.want)
		}
	}
}

func TestContainsServer_MatchesHostAndPort(t *testing.T) {
	existing := []config.Server{
		{Host: "a.com", Port: 22},
		{Host: "b.com", Port: 2222},
	}

	cases := []struct {
		s    config.Server
		want bool
	}{
		{config.Server{Host: "a.com", Port: 22}, true},
		{config.Server{Host: "a.com", Port: 2222}, false}, // same host, different port = different server
		{config.Server{Host: "c.com", Port: 22}, false},
		{config.Server{Host: "b.com", Port: 2222}, true},
	}

	for _, tt := range cases {
		if got := containsServer(existing, tt.s); got != tt.want {
			t.Errorf("containsServer(%+v) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestIsStandardBinPath(t *testing.T) {
	t.Setenv("HOME", "/home/test")
	tests := []struct {
		path string
		want bool
	}{
		{"/usr/local/bin/cpssh", true},
		{"/opt/homebrew/bin/cpssh", true},
		{"/usr/bin/cpssh", true},
		{"/home/test/bin/cpssh", true},
		{"/home/test/.local/bin/cpssh", true},
		{"/tmp/cpssh", false},
		{"/Users/test/Downloads/cpssh", false},
	}
	for _, tt := range tests {
		if got := isStandardBinPath(tt.path); got != tt.want {
			t.Errorf("isStandardBinPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestIsValidPort(t *testing.T) {
	tests := []struct {
		p    int
		want bool
	}{
		{0, false},
		{-1, false},
		{1, true},
		{22, true},
		{65535, true},
		{65536, false},
		{100000, false},
	}
	for _, tt := range tests {
		if got := isValidPort(tt.p); got != tt.want {
			t.Errorf("isValidPort(%d) = %v, want %v", tt.p, got, tt.want)
		}
	}
}

func TestPromptInt(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		min, max   int
		defaultVal *int
		want       int
	}{
		{name: "valid input", input: "42\n", min: 1, max: 100, want: 42},
		{name: "boundary low", input: "1\n", min: 1, max: 100, want: 1},
		{name: "boundary high", input: "100\n", min: 1, max: 100, want: 100},
		{name: "empty with default", input: "\n", min: 1, max: 100, defaultVal: intPtr(22), want: 22},
		{name: "retries past invalid", input: "abc\n-5\n200\n50\n", min: 1, max: 100, want: 50},
		{name: "retries past empty when no default", input: "\n7\n", min: 1, max: 100, want: 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(tt.input))
			got := promptInt(r, "test: ", tt.min, tt.max, tt.defaultVal)
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func intPtr(i int) *int { return &i }

func TestShellSingleQuote(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"simple", `'simple'`},
		{"with space", `'with space'`},
		{`it's`, `'it'\''s'`},
		{"", `''`},
		{`$HOME/.cpssh`, `'$HOME/.cpssh'`}, // $ stays literal — no expansion inside single quotes
		{`a;rm -rf /b`, `'a;rm -rf /b'`},   // semicolons inside single quotes are literal
		{`"quoted"`, `'"quoted"'`},          // double quotes are fine inside single quotes
	}
	for _, tt := range tests {
		if got := shellSingleQuote(tt.in); got != tt.want {
			t.Errorf("shellSingleQuote(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
