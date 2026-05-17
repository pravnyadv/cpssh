package sync

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/pravnyadv/cpssh/internal/config"
)

var mu sync.Mutex

// SyncToAll syncs imageData to all configured servers in parallel.
// Returns the remote path of the synced file (from first successful server).
// A mutex ensures only one sync runs at a time — prevents symlink race on rapid copies.
func SyncToAll(cfg *config.Config, imageData []byte) string {
	mu.Lock()
	defer mu.Unlock()

	filename := config.NextImageName()
	data := maybeCompressBytes(cfg, imageData)

	var (
		wg         sync.WaitGroup
		remotePath string
		setOnce    sync.Once
	)
	for _, srv := range cfg.Servers {
		wg.Add(1)
		go func(s config.Server) {
			defer wg.Done()
			if err := syncToServer(s, data, filename, cfg.Settings.KeepLastNFiles); err != nil {
				log.Printf("sync: [%s] error: %v", s.Host, err)
			} else {
				log.Printf("sync: [%s] ok → %s/%s", s.Host, s.SyncPath, filename)
				setOnce.Do(func() { remotePath = s.SyncPath + "/" + filename })
			}
		}(srv)
	}
	wg.Wait()
	return remotePath
}

// maybeCompressBytes returns imageData unchanged when below the threshold —
// the common case for screenshots — avoiding a write/read round-trip through
// the filesystem. When compression IS needed, sips/convert require file inputs,
// so we round-trip then.
func maybeCompressBytes(cfg *config.Config, imageData []byte) []byte {
	if int64(len(imageData)) <= int64(cfg.Settings.CompressAboveKB)*1024 {
		return imageData
	}

	in, err := os.CreateTemp("", "cpssh-in-*.png")
	if err != nil {
		return imageData
	}
	inPath := in.Name()
	defer os.Remove(inPath)
	if _, err := in.Write(imageData); err != nil {
		in.Close()
		return imageData
	}
	in.Close()

	outPath := inPath[:len(inPath)-4] + "_c.png"
	defer os.Remove(outPath)

	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("sips", "-s", "format", "png", "--resampleHeightWidthMax", "2000", inPath, "--out", outPath)
	} else {
		cmd = exec.Command("convert", inPath, "-resize", "2000x2000>", outPath)
	}
	if err := cmd.Run(); err != nil {
		return imageData
	}
	compressed, err := os.ReadFile(outPath)
	if err != nil {
		return imageData
	}
	return compressed
}

// sshArgs returns common SSH flags: identity, ControlMaster (reuses connections
// across calls so the 2nd+ syncs in a session are near-instant), BatchMode so a
// missing key fails fast instead of hanging on a passphrase prompt, and no host
// key prompt on first connect.
func sshArgs(s config.Server) []string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = os.Getenv("HOME")
	}
	args := []string{
		"-i", s.SSHKey,
		"-o", "BatchMode=yes",
		"-o", "ControlMaster=auto",
		"-o", fmt.Sprintf("ControlPath=%s/.ssh/cm_%%r@%%h:%%p", home),
		"-o", "ControlPersist=10m",
		"-o", "StrictHostKeyChecking=accept-new",
	}
	if s.Port != 0 {
		args = append(args, "-p", fmt.Sprintf("%d", s.Port))
	}
	return args
}

// WarmUp pre-establishes the ControlMaster socket for all servers so the
// first real sync reuses an existing connection instead of paying the handshake cost.
func WarmUp(cfg *config.Config) {
	for _, s := range cfg.Servers {
		go func(srv config.Server) {
			args := append(sshArgs(srv), fmt.Sprintf("%s@%s", srv.User, srv.Host), "true")
			cmd := exec.Command("ssh", args...)
			if err := cmd.Run(); err != nil {
				log.Printf("sync: warmup [%s] failed: %v", srv.Host, err)
			} else {
				log.Printf("sync: [%s] connection warmed up", srv.Host)
			}
		}(s)
	}
}

func syncToServer(s config.Server, data []byte, filename string, keepN int) error {
	remoteCmd := buildRemoteCmd(s.SyncPath, filename, keepN)
	args := append(sshArgs(s), fmt.Sprintf("%s@%s", s.User, s.Host), remoteCmd)
	run := func() error {
		cmd := exec.Command("ssh", args...)
		cmd.Stdin = bytes.NewReader(data)
		return cmd.Run()
	}
	if err := run(); err != nil {
		time.Sleep(time.Second)
		return run()
	}
	return nil
}

// buildRemoteCmd assembles the SSH-side shell command that writes the new
// image, repoints the latest.png symlink, and prunes old files.
//
// `xargs -r` is GNU's "skip when input is empty" flag; FreeBSD/macOS xargs
// accept it as a documented no-op (it never invokes the utility on empty input
// anyway). Without -r, GNU xargs would run `rm -f` with no operands, which
// errors out and makes the whole && chain fail — silently breaking the
// clipboard text reference on every fresh Linux server.
func buildRemoteCmd(syncPath, filename string, keepN int) string {
	return fmt.Sprintf(
		`mkdir -p "%s" && cat > "%s/%s" && cd "%s" && ln -sf "%s" latest.png && ls -t *.png 2>/dev/null | tail -n +%d | xargs -r rm -f`,
		syncPath, syncPath, filename, syncPath, filename, keepN+1,
	)
}
