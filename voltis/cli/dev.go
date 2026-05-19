package cli

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/voltis/voltis/voltis/runtime"
)

func runDev(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("dev", flag.ContinueOnError)
	addr := fs.String("addr", "", "listen address")
	if err := fs.Parse(args); err != nil {
		return err
	}

	root, err := os.Getwd()
	if err != nil {
		return err
	}

	cfg, err := runtime.LoadConfig(filepath.Join(root, "voltis.config.json"))
	if err != nil {
		cfg = runtime.DefaultConfig()
	}
	if *addr != "" {
		cfg.HTTP.Addr = *addr
	}

	viteURL, stopVite, err := startVite(ctx, filepath.Join(root, cfg.AppDir), cfg.Dev.VitePort)
	if err != nil {
		return err
	}
	defer stopVite()

	return runGoServer(ctx, root, cfg.HTTP.Addr, viteURL)
}

func startVite(ctx context.Context, appRoot string, port int) (viteURL string, stop func(), err error) {
	cmd := exec.CommandContext(ctx, "npm", "run", "dev", "--", "--host", "127.0.0.1", "--port", fmt.Sprintf("%d", port), "--strictPort")
	cmd.Dir = appRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return "", func() {}, err
	}

	stop = func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}

	viteURL = fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, viteURL+"/", nil)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 500 {
				return viteURL, stop, nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	stop()
	return "", func() {}, fmt.Errorf("vite dev server not ready at %s", viteURL)
}

func runGoServer(ctx context.Context, root string, addr string, devProxy string) error {
	cmd := exec.CommandContext(ctx, "go", "run", "./server", "-addr", addr, "-dev-proxy", devProxy)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
