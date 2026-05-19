package actions

import (
	"errors"
	"strings"

	"github.com/Gsykes27/voltis/voltis/runtime"
)

func init() {
	Registry.Register("CreateTicket", func(ctx runtime.ActionCtx, data map[string]any) (any, error) {
		title, _ := runtime.GetString(data, "title")
		priority, _ := runtime.GetString(data, "priority")
		if strings.TrimSpace(title) == "" {
			return nil, errors.New("missing title")
		}
		if priority == "" {
			priority = "normal"
		}
		t := ctx.Server.TicketCreate(title, priority)
		ctx.Server.Publish(ctx.Context, "tickets", map[string]any{"type": "created", "ticket": t})
		return t, nil
	})

	Registry.Register("ListTickets", func(ctx runtime.ActionCtx, data map[string]any) (any, error) {
		return ctx.Server.TicketList(), nil
	})

	Registry.Register("ResolveTicket", func(ctx runtime.ActionCtx, data map[string]any) (any, error) {
		id, ok := runtime.GetInt64(data, "id")
		if !ok || id <= 0 {
			return nil, errors.New("invalid id")
		}
		t, err := ctx.Server.TicketResolve(id)
		if err != nil {
			return nil, err
		}
		ctx.Server.Publish(ctx.Context, "tickets", map[string]any{"type": "resolved", "ticket": t})
		return t, nil
	})
}
