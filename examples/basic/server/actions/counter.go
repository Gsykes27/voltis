package actions

import "github.com/voltis/voltis/voltis/runtime"

func init() {
	Registry.Register("GetCounter", func(ctx runtime.ActionCtx, data map[string]any) (any, error) {
		return ctx.Server.CounterGet(), nil
	})
	Registry.Register("IncrementCounter", func(ctx runtime.ActionCtx, data map[string]any) (any, error) {
		v := ctx.Server.CounterIncrement()
		ctx.Server.Publish(ctx.Context, "counter", map[string]any{"value": v})
		return v, nil
	})
}

