package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func runCreate(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return usage("create requires a target")
	}

	switch args[0] {
	case "app":
		fs := flag.NewFlagSet("create app", flag.ContinueOnError)
		mod := fs.String("module", "example.com/voltis-app", "go module path")
		noInstall := fs.Bool("no-install", false, "skip npm install")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		rest := fs.Args()
		if len(rest) != 1 {
			return errors.New("usage: voltis create app -module <module> <dir>")
		}
		return createApp(ctx, rest[0], *mod, *noInstall)
	default:
		return fmt.Errorf("unknown create target: %s", args[0])
	}
}

func createApp(ctx context.Context, dir string, module string, noInstall bool) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	write := func(rel string, data string) error {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if module != "" {
			data = strings.ReplaceAll(data, "example.com/voltis-app", module)
		}
		return os.WriteFile(p, []byte(data), 0o644)
	}

	if err := write("voltis.config.json", tmplVoltisConfig); err != nil {
		return err
	}
	if err := write(".gitignore", tmplGitIgnore); err != nil {
		return err
	}
	if err := write("go.mod", renderAppGoMod(module, findVoltisRepoRoot())); err != nil {
		return err
	}
	if err := write("app/package.json", tmplAppPackageJSON); err != nil {
		return err
	}
	if err := write("app/vite.config.ts", tmplViteConfig); err != nil {
		return err
	}
	if err := write("app/index.html", tmplIndexHTML); err != nil {
		return err
	}
	if err := write("app/tsconfig.json", tmplTSConfig); err != nil {
		return err
	}
	if err := write("app/src/main.tsx", tmplMainTSX); err != nil {
		return err
	}
	if err := write("app/src/router.tsx", tmplRouterTSX); err != nil {
		return err
	}
	if err := write("app/src/voltis.ts", tmplVoltisTS); err != nil {
		return err
	}
	if err := write("app/src/routes/index.tsx", tmplHomeRoute); err != nil {
		return err
	}
	if err := write("app/src/styles.css", tmplStylesCSS); err != nil {
		return err
	}
	if err := write("server/main.go", tmplServerMainGo); err != nil {
		return err
	}
	if err := write("server/actions/registry.go", tmplServerRegistryGo); err != nil {
		return err
	}
	if err := write("server/actions/counter.go", tmplServerCounterActionGo); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, "public"), 0o755); err != nil {
		return err
	}
	if err := write("public/.gitkeep", ""); err != nil {
		return err
	}

	tidy := exec.CommandContext(ctx, "go", "mod", "tidy")
	tidy.Dir = dir
	tidy.Stdout = os.Stdout
	tidy.Stderr = os.Stderr
	if err := tidy.Run(); err != nil {
		return err
	}

	if !noInstall {
		install := exec.CommandContext(ctx, "npm", "install")
		install.Dir = filepath.Join(dir, "app")
		install.Stdout = os.Stdout
		install.Stderr = os.Stderr
		if err := install.Run(); err != nil {
			return err
		}
	}

	return nil
}

func renderAppGoMod(module string, voltisLocalPath string) string {
	var b strings.Builder
	if module == "" {
		module = "example.com/voltis-app"
	}
	b.WriteString("module ")
	b.WriteString(module)
	b.WriteString("\n\ngo 1.22\n\nrequire github.com/Gsykes27/voltis v0.0.0\n")
	if voltisLocalPath != "" && !strings.ContainsAny(voltisLocalPath, " \t") {
		b.WriteString("\nreplace github.com/Gsykes27/voltis => ")
		b.WriteString(filepath.ToSlash(voltisLocalPath))
		b.WriteString("\n")
	}
	return b.String()
}

func findVoltisRepoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := wd
	for i := 0; i < 10; i++ {
		p := filepath.Join(dir, "go.mod")
		b, err := os.ReadFile(p)
		if err == nil && strings.Contains(string(b), "module github.com/Gsykes27/voltis") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
