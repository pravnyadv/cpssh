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

	localFile, err := saveTempFile(imageData)
	if err != nil {
		log.Printf("sync: failed to save temp file: %v", err)
		return ""
	}

	localFile = maybeCompress(cfg, localFile)
	defer os.Remove(localFile)

	data, err := os.ReadFile(localFile)
	if err != nil {
		log.Printf("sync: failed to read temp file: %v", err)
		return ""
	}

	var (
		wg         sync.WaitGroup
		remotePath string
		setOnce    sync.Once
	)
	for _, srv := range cfg.Servers {
		wg.Add(1)
		go func(s config.Server) {
			defer wg.Done()
			if err := syncToServer(s, data, filename); err != nil {
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

func saveTempFile(data []byte) (string, error) {
	f, err := os.CreateTemp("", "cpssh-*.png")
	if err != nil {
		return "", err
	}
	_, werr := f.Write(data)
	f.Close()
	if werr != nil {
		os.Remove(f.Name())
		return "", werr
	}
	return f.Name(), nil
}

func maybeCompress(cfg *config.Config, path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return path
	}
	if info.Size() <= int64(cfg.Settings.CompressAboveKB)*1024 {
		return path
	}

	out := path[:len(path)-4] + "_compressed.png"
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("sips", "-s", "format", "png", "--resampleHeightWidthMax", "2000", path, "--out", out)
	} else {
		cmd = exec.Command("convert", path, "-resize", "2000x2000>", out)
	}
	if err := cmd.Run(); err != nil {
		return path
	}
	os.Remove(path)
	return out
}

// sshArgs returns common SSH flags: identity, ControlMaster (reuses connections
// across calls so the 2nd+ syncs in a session are near-instant), and no host
// key prompt on first connect.
func sshArgs(s config.Server) []string {
	return []string{
		"-i", s.SSHKey,
		"-o", "ControlMaster=auto",
		"-o", fmt.Sprintf("ControlPath=%s/.ssh/cm_%%r@%%h:%%p", os.Getenv("HOME")),
		"-o", "ControlPersist=10m",
		"-o", "StrictHostKeyChecking=accept-new",
	}
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

func syncToServer(s config.Server, data []byte, filename string) error {
	remoteCmd := fmt.Sprintf(
		`cat > "%s/%s" && cd "%s" && ln -sf "%s" latest.png && ls -t *.png 2>/dev/null | tail -n +11 | xargs rm -f`,
		s.SyncPath, filename, s.SyncPath, filename,
	)
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
