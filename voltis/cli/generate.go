package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Gsykes27/voltis/voltis/runtime"
)

func runGenerate(ctx context.Context, args []string) error {
	_ = ctx
	if len(args) < 2 {
		return errors.New("usage: voltis generate (route|action) <value>")
	}

	root, err := os.Getwd()
	if err != nil {
		return err
	}

	cfg, err := runtime.LoadConfig(filepath.Join(root, "voltis.config.json"))
	if err != nil {
		cfg = runtime.DefaultConfig()
	}

	switch args[0] {
	case "route":
		return genRoute(filepath.Join(root, cfg.AppDir), args[1])
	case "action":
		return genAction(filepath.Join(root, "server"), args[1])
	default:
		return fmt.Errorf("unknown generate target: %s", args[0])
	}
}

func genRoute(appRoot string, p string) error {
	p = strings.TrimSpace(p)
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return errors.New("route path is empty")
	}
	if strings.HasSuffix(p, "/") {
		p += "index"
	}
	p = strings.ReplaceAll(p, "\\", "/")

	if !strings.HasSuffix(p, ".tsx") {
		p += ".tsx"
	}

	out := filepath.Join(appRoot, "src", "routes", filepath.FromSlash(p))
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(out); err == nil {
		return fmt.Errorf("route already exists: %s", out)
	}

	name := routeComponentName(p)
	code := fmt.Sprintf(`import React from "react"

export default function %s() {
  return (
    <div className="container">
      <h1>%s</h1>
      <p className="muted">Nova rota criada com Voltis.</p>
    </div>
  )
}
`, name, strings.TrimSuffix(strings.TrimSuffix(p, ".tsx"), "/index"))

	return os.WriteFile(out, []byte(code), 0o644)
}

func routeComponentName(file string) string {
	file = strings.TrimSuffix(filepath.ToSlash(file), ".tsx")
	file = strings.TrimSuffix(file, "/index")
	file = strings.ReplaceAll(file, "/", "_")
	file = regexp.MustCompile(`[^A-Za-z0-9_]`).ReplaceAllString(file, "_")
	if file == "" {
		return "Page"
	}
	parts := strings.Split(file, "_")
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, "")
}

func genAction(serverRoot string, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("action name is empty")
	}
	if !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(name) {
		return errors.New("action name must be a valid identifier, e.g. CreateTicket")
	}

	actionsDir := filepath.Join(serverRoot, "actions")
	if err := os.MkdirAll(actionsDir, 0o755); err != nil {
		return err
	}

	registryFile := filepath.Join(actionsDir, "registry.go")
	if _, err := os.Stat(registryFile); os.IsNotExist(err) {
		reg := `package actions

import "github.com/Gsykes27/voltis/voltis/runtime"

var Registry = runtime.NewActionRegistry()
`
		if err := os.WriteFile(registryFile, []byte(reg), 0o644); err != nil {
			return err
		}
	}

	out := filepath.Join(actionsDir, toSnake(name)+".go")
	if _, err := os.Stat(out); err == nil {
		return fmt.Errorf("action already exists: %s", out)
	}

	code := fmt.Sprintf(`package actions

import "github.com/Gsykes27/voltis/voltis/runtime"

func init() {
  Registry.Register(%q, func(ctx runtime.ActionCtx, data map[string]any) (any, error) {
    return map[string]any{"ok": true}, nil
  })
}
`, name)

	return os.WriteFile(out, []byte(code), 0o644)
}

func toSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}
