package actions

import (
	"errors"
	"strings"

	"github.com/Gsykes27/voltis/voltis/runtime"
)

func init() {
	Registry.Register("SendChatMessage", func(ctx runtime.ActionCtx, data map[string]any) (any, error) {
		room, _ := runtime.GetString(data, "room")
		author, _ := runtime.GetString(data, "author")
		text, _ := runtime.GetString(data, "text")
		if room == "" {
			room = "support"
		}
		if strings.TrimSpace(author) == "" {
			author = "Anonymous"
		}
		if strings.TrimSpace(text) == "" {
			return nil, errors.New("missing text")
		}
		msg := ctx.Server.ChatAppend(room, author, text)
		ctx.Server.Publish(ctx.Context, "chat:"+room, map[string]any{"type": "message", "message": msg})
		return msg, nil
	})

	Registry.Register("ListChatMessages", func(ctx runtime.ActionCtx, data map[string]any) (any, error) {
		room, _ := runtime.GetString(data, "room")
		if room == "" {
			room = "support"
		}
		return ctx.Server.ChatList(room), nil
	})
}
