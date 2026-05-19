package runtime

import (
	"context"
	"encoding/json"
	"sync"

	"nhooyr.io/websocket"
)

type wsHub struct {
	mu       sync.RWMutex
	channels map[string]map[*wsConn]struct{}
}

type wsConn struct {
	c    *websocket.Conn
	mu   sync.Mutex
	subs map[string]struct{}
}

func newWSHub() *wsHub {
	return &wsHub{channels: map[string]map[*wsConn]struct{}{}}
}

func (h *wsHub) AddConn(c *websocket.Conn) *wsConn {
	wc := &wsConn{c: c, subs: map[string]struct{}{}}
	return wc
}

func (h *wsHub) RemoveConn(wc *wsConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range wc.subs {
		if set, ok := h.channels[ch]; ok {
			delete(set, wc)
			if len(set) == 0 {
				delete(h.channels, ch)
			}
		}
	}
}

func (h *wsHub) Subscribe(wc *wsConn, ch string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	wc.subs[ch] = struct{}{}
	set, ok := h.channels[ch]
	if !ok {
		set = map[*wsConn]struct{}{}
		h.channels[ch] = set
	}
	set[wc] = struct{}{}
}

func (h *wsHub) Unsubscribe(wc *wsConn, ch string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(wc.subs, ch)
	if set, ok := h.channels[ch]; ok {
		delete(set, wc)
		if len(set) == 0 {
			delete(h.channels, ch)
		}
	}
}

func (h *wsHub) Publish(ctx context.Context, ch string, payload any) {
	msg := map[string]any{"t": "event", "ch": ch, "data": payload}
	b, _ := json.Marshal(msg)

	h.mu.RLock()
	set := h.channels[ch]
	conns := make([]*wsConn, 0, len(set))
	for wc := range set {
		conns = append(conns, wc)
	}
	h.mu.RUnlock()

	for _, wc := range conns {
		wc.write(ctx, websocket.MessageText, b)
	}
}

func (wc *wsConn) write(ctx context.Context, typ websocket.MessageType, b []byte) {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	_ = wc.c.Write(ctx, typ, b)
}

