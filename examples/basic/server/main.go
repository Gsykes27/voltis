package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"

	"github.com/Gsykes27/voltis/examples/basic/server/actions"
	"github.com/Gsykes27/voltis/voltis/runtime"
)

func main() {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	addr := fs.String("addr", "", "listen address")
	devProxy := fs.String("dev-proxy", "", "vite dev proxy url")
	_ = fs.Parse(os.Args[1:])

	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	cfg, err := runtime.LoadConfig(filepath.Join(root, "voltis.config.json"))
	if err != nil {
		cfg = runtime.DefaultConfig()
	}
	if *addr != "" {
		cfg.HTTP.Addr = *addr
	}

	s, err := runtime.NewServer(root, cfg, runtime.ServerOptions{
		DevProxyURL: *devProxy,
		Actions:     actions.Registry,
	})
	if err != nil {
		panic(err)
	}

	if err := s.Serve(context.Background()); err != nil {
		panic(err)
	}
}
