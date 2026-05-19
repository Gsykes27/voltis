package builder

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type ViteOptions struct {
	RootDir string
	AppDir  string
	DistDir string
}

func ViteBuild(ctx context.Context, opt ViteOptions) error {
	appRoot := filepath.Join(opt.RootDir, opt.AppDir)
	if _, err := os.Stat(filepath.Join(appRoot, "package.json")); err != nil {
		return fmt.Errorf("vite appDir missing package.json: %s", appRoot)
	}

	cmd := exec.CommandContext(ctx, "npm", "run", "build")
	cmd.Dir = appRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	distRoot := filepath.Join(opt.RootDir, opt.DistDir)
	if _, err := os.Stat(distRoot); err != nil {
		return errors.New("build produced no dist directory; check vite config outDir")
	}
	return nil
}

