package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/voltis/voltis/voltis/builder"
	"github.com/voltis/voltis/voltis/runtime"
)

func runBuild(ctx context.Context, args []string) error {
	_ = args
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	cfg, err := runtime.LoadConfig(filepath.Join(root, "voltis.config.json"))
	if err != nil {
		cfg = runtime.DefaultConfig()
	}
	if err := builder.ViteBuild(ctx, builder.ViteOptions{
		RootDir: root,
		AppDir:  cfg.AppDir,
		DistDir: cfg.DistDir,
	}); err != nil {
		return err
	}

	outDir := filepath.Join(root, cfg.DistDir, "server")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	out := filepath.Join(outDir, "voltis-server")
	if isWindows() {
		out += ".exe"
	}

	tidy := exec.CommandContext(ctx, "go", "mod", "tidy")
	tidy.Dir = root
	tidy.Stdout = os.Stdout
	tidy.Stderr = os.Stderr
	if err := tidy.Run(); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "go", "build", "-o", out, "./server")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func isWindows() bool {
	return os.PathSeparator == '\\'
}
