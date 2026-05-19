package cli

import (
	"context"
	"flag"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Gsykes27/voltis/voltis/runtime"
)

func runStart(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
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

	bin := filepath.Join(root, cfg.DistDir, "server", "voltis-server")
	if isWindows() {
		bin += ".exe"
	}
	if _, err := os.Stat(bin); err == nil {
		cmd := exec.CommandContext(ctx, bin, "-addr", cfg.HTTP.Addr)
		cmd.Dir = root
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	cmd := exec.CommandContext(ctx, "go", "run", "./server", "-addr", cfg.HTTP.Addr)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
