package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
)

func Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usage("")
	}

	switch args[0] {
	case "help", "-h", "--help":
		return usage("")
	case "create":
		return runCreate(ctx, args[1:])
	case "generate", "g":
		return runGenerate(ctx, args[1:])
	case "dev":
		return runDev(ctx, args[1:])
	case "build":
		return runBuild(ctx, args[1:])
	case "start":
		return runStart(ctx, args[1:])
	case "doctor", "analyze", "deploy", "cluster", "worker":
		return errors.New("not implemented in MVP")
	default:
		return usage(fmt.Sprintf("unknown command: %s", args[0]))
	}
}

func usage(prefix string) error {
	if prefix != "" {
		fmt.Fprintln(os.Stderr, prefix)
		fmt.Fprintln(os.Stderr)
	}
	fmt.Fprintln(os.Stderr, "Voltis - realtime-first fullstack framework powered by Go.")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  voltis create app -module <module> [-no-install] <dir>")
	fmt.Fprintln(os.Stderr, "  voltis generate route <path>")
	fmt.Fprintln(os.Stderr, "  voltis generate action <Name>")
	fmt.Fprintln(os.Stderr, "  voltis dev [-addr :3000]")
	fmt.Fprintln(os.Stderr, "  voltis build")
	fmt.Fprintln(os.Stderr, "  voltis start [-addr :3000]")
	return nil
}
