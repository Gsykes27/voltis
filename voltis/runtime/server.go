package runtime

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

type ServerOptions struct {
	DevProxyURL string
	Actions     *ActionRegistry
}

type Server struct {
	cfg     Config
	rootDir string
	distDir string

	devProxy *DevProxy

	actions *ActionRegistry

	db *memDB
	ws *wsHub

	replayMu sync.Mutex
	replay   map[string]time.Time

	rateMu sync.Mutex
	rate   map[string]*tokenBucket
}

func NewServer(rootDir string, cfg Config, opt ServerOptions) (*Server, error) {
	dist := filepath.Join(rootDir, cfg.DistDir)
	s := &Server{
		cfg:     cfg,
		rootDir: rootDir,
		distDir: dist,
		actions: opt.Actions,
		db:      newMemDB(),
		ws:      newWSHub(),
		replay:  map[string]time.Time{},
		rate:    map[string]*tokenBucket{},
	}
	if s.actions == nil {
		s.actions = NewActionRegistry()
	}

	if opt.DevProxyURL != "" {
		p, err := NewDevProxy(opt.DevProxyURL)
		if err != nil {
			return nil, err
		}
		s.devProxy = p
	}
	return s, nil
}

func (s *Server) Serve(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/__vlt/sdk.js", s.handleSDK)
	mux.HandleFunc("/__vlt/actions/", s.handleAction)
	mux.HandleFunc("/__vlt/ws", s.handleWS)

	mux.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir(filepath.Join(s.rootDir, "public")))))

	if s.devProxy != nil {
		mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.ensureSession(w, r)
			s.devProxy.ServeHTTP(w, r)
		}))
	} else {
		mux.HandleFunc("/", s.handleStaticOrIndex)
	}

	srv := &http.Server{
		Addr:              s.cfg.HTTP.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func (s *Server) handleStaticOrIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.ensureSession(w, r)

	clientRoot := filepath.Join(s.distDir, "client")
	p := filepath.Clean(filepath.Join(clientRoot, filepath.FromSlash(strings.TrimPrefix(r.URL.Path, "/"))))
	if !strings.HasPrefix(p, clientRoot) {
		http.NotFound(w, r)
		return
	}

	if st, err := os.Stat(p); err == nil && !st.IsDir() {
		http.ServeFile(w, r, p)
		return
	}

	http.ServeFile(w, r, filepath.Join(clientRoot, "index.html"))
}

func (s *Server) handleAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.ensureSession(w, r)
	if !s.allowSameOrigin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if !s.allowRate(r) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
		return
	}
	if !s.allowCSRF(r) {
		http.Error(w, "csrf", http.StatusForbidden)
		return
	}
	sid, ok := s.verifyActCookie(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/__vlt/actions/")
	name, _ = url.PathUnescape(name)
	if name == "" {
		http.Error(w, "missing action name", http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req struct {
		Data  map[string]any `json:"data"`
		TS    int64          `json:"ts"`
		Nonce string         `json:"nonce"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Nonce == "" || req.TS == 0 {
		http.Error(w, "missing nonce/ts", http.StatusBadRequest)
		return
	}
	if !s.allowFresh(req.TS) {
		http.Error(w, "expired", http.StatusForbidden)
		return
	}
	if !s.allowReplay(sid, name, req.Nonce) {
		http.Error(w, "replay", http.StatusForbidden)
		return
	}

	fn, ok := s.actions.Get(name)
	if !ok {
		http.Error(w, "action not found", http.StatusNotFound)
		return
	}
	result, err := fn(ActionCtx{Context: r.Context(), Server: s}, req.Data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"result": result})
}

type memDB struct {
	counter *memCounter
	tickets *memTickets
	chat    *memChat
}

type memCounter struct {
	mu sync.Mutex
	v  int64
}

func newMemDB() *memDB {
	return &memDB{
		counter: &memCounter{},
		tickets: &memTickets{byID: map[int64]Ticket{}},
		chat:    &memChat{rooms: map[string][]ChatMessage{}},
	}
}

func (db *memDB) CounterIncrement() int64 {
	db.counter.mu.Lock()
	defer db.counter.mu.Unlock()
	db.counter.v++
	return db.counter.v
}

func (db *memDB) CounterGet() int64 {
	db.counter.mu.Lock()
	defer db.counter.mu.Unlock()
	return db.counter.v
}

type Ticket struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Priority  string `json:"priority"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

type memTickets struct {
	mu     sync.Mutex
	nextID int64
	byID   map[int64]Ticket
	order  []int64
}

func (db *memDB) TicketCreate(title, priority string) Ticket {
	now := time.Now().UnixMilli()
	db.tickets.mu.Lock()
	defer db.tickets.mu.Unlock()
	db.tickets.nextID++
	t := Ticket{
		ID:        db.tickets.nextID,
		Title:     title,
		Priority:  priority,
		Status:    "open",
		CreatedAt: now,
		UpdatedAt: now,
	}
	db.tickets.byID[t.ID] = t
	db.tickets.order = append(db.tickets.order, t.ID)
	return t
}

func (db *memDB) TicketList() []Ticket {
	db.tickets.mu.Lock()
	defer db.tickets.mu.Unlock()
	out := make([]Ticket, 0, len(db.tickets.order))
	for i := len(db.tickets.order) - 1; i >= 0; i-- {
		id := db.tickets.order[i]
		out = append(out, db.tickets.byID[id])
	}
	return out
}

func (db *memDB) TicketResolve(id int64) (Ticket, error) {
	now := time.Now().UnixMilli()
	db.tickets.mu.Lock()
	defer db.tickets.mu.Unlock()
	t, ok := db.tickets.byID[id]
	if !ok {
		return Ticket{}, errors.New("ticket not found")
	}
	t.Status = "resolved"
	t.UpdatedAt = now
	db.tickets.byID[id] = t
	return t, nil
}

type ChatMessage struct {
	ID        int64  `json:"id"`
	Room      string `json:"room"`
	Author    string `json:"author"`
	Text      string `json:"text"`
	CreatedAt int64  `json:"createdAt"`
}

type memChat struct {
	mu     sync.Mutex
	nextID int64
	rooms  map[string][]ChatMessage
}

func (db *memDB) ChatAppend(room, author, text string) ChatMessage {
	now := time.Now().UnixMilli()
	db.chat.mu.Lock()
	defer db.chat.mu.Unlock()
	db.chat.nextID++
	msg := ChatMessage{
		ID:        db.chat.nextID,
		Room:      room,
		Author:    author,
		Text:      text,
		CreatedAt: now,
	}
	db.chat.rooms[room] = append(db.chat.rooms[room], msg)
	if len(db.chat.rooms[room]) > 100 {
		db.chat.rooms[room] = db.chat.rooms[room][len(db.chat.rooms[room])-100:]
	}
	return msg
}

func (db *memDB) ChatList(room string) []ChatMessage {
	db.chat.mu.Lock()
	defer db.chat.mu.Unlock()
	src := db.chat.rooms[room]
	out := make([]ChatMessage, len(src))
	copy(out, src)
	return out
}

func (s *Server) CounterIncrement() int64 { return s.db.CounterIncrement() }
func (s *Server) CounterGet() int64       { return s.db.CounterGet() }
func (s *Server) TicketCreate(title, priority string) Ticket {
	return s.db.TicketCreate(title, priority)
}
func (s *Server) TicketList() []Ticket                   { return s.db.TicketList() }
func (s *Server) TicketResolve(id int64) (Ticket, error) { return s.db.TicketResolve(id) }
func (s *Server) ChatAppend(room, author, text string) ChatMessage {
	return s.db.ChatAppend(room, author, text)
}
func (s *Server) ChatList(room string) []ChatMessage { return s.db.ChatList(room) }
func (s *Server) Publish(ctx context.Context, ch string, payload any) {
	s.ws.Publish(ctx, ch, payload)
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	s.ensureSession(w, r)
	if !s.allowSameOrigin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if _, ok := s.verifyActCookie(r); !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	wc := s.ws.AddConn(c)
	defer func() {
		s.ws.RemoveConn(wc)
		_ = c.Close(websocket.StatusNormalClosure, "")
	}()

	ctx := r.Context()
	for {
		_, data, err := c.Read(ctx)
		if err != nil {
			return
		}

		var msg struct {
			T    string          `json:"t"`
			Ch   string          `json:"ch"`
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		switch msg.T {
		case "ping":
			wc.write(ctx, websocket.MessageText, []byte(`{"t":"pong"}`))
		case "sub":
			if msg.Ch != "" {
				s.ws.Subscribe(wc, msg.Ch)
				wc.write(ctx, websocket.MessageText, []byte(`{"t":"sub_ack"}`))
			}
		case "unsub":
			if msg.Ch != "" {
				s.ws.Unsubscribe(wc, msg.Ch)
				wc.write(ctx, websocket.MessageText, []byte(`{"t":"unsub_ack"}`))
			}
		case "pub":
			if msg.Ch == "" {
				continue
			}
			var payload any
			if len(msg.Data) > 0 {
				_ = json.Unmarshal(msg.Data, &payload)
			}
			s.ws.Publish(ctx, msg.Ch, payload)
		default:
		}
	}
}

func (s *Server) allowSameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	host := r.Host
	if strings.HasPrefix(origin, "http://"+host) || strings.HasPrefix(origin, "https://"+host) {
		return true
	}
	return false
}

func (s *Server) allowRate(r *http.Request) bool {
	ip := clientIP(r)
	s.rateMu.Lock()
	defer s.rateMu.Unlock()

	b, ok := s.rate[ip]
	if !ok {
		b = newTokenBucket(50, 100, time.Second)
		s.rate[ip] = b
	}
	return b.Allow()
}

type tokenBucket struct {
	capacity int
	tokens   int
	refill   int
	period   time.Duration
	last     time.Time
}

func newTokenBucket(refill, capacity int, period time.Duration) *tokenBucket {
	return &tokenBucket{
		capacity: capacity,
		tokens:   capacity,
		refill:   refill,
		period:   period,
		last:     time.Now(),
	}
}

func (b *tokenBucket) Allow() bool {
	now := time.Now()
	if now.Sub(b.last) >= b.period {
		n := int(now.Sub(b.last) / b.period)
		b.tokens += n * b.refill
		if b.tokens > b.capacity {
			b.tokens = b.capacity
		}
		b.last = b.last.Add(time.Duration(n) * b.period)
	}
	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

func (s *Server) handleSDK(w http.ResponseWriter, r *http.Request) {
	s.ensureSession(w, r)
	w.Header().Set("content-type", "application/javascript; charset=utf-8")
	w.Header().Set("cache-control", "no-store")
	w.Header().Set("x-content-type-options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, sdkJS)
}

func (s *Server) ensureSession(w http.ResponseWriter, r *http.Request) {
	const csrfCookie = "__vlt_csrf"
	const actCookie = "__vlt_act"

	secure := r.TLS != nil
	if _, err := r.Cookie(csrfCookie); err != nil {
		v := randB64(24)
		http.SetCookie(w, &http.Cookie{
			Name:     csrfCookie,
			Value:    v,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Secure:   secure,
			HttpOnly: false,
		})
	}
	if _, err := r.Cookie(actCookie); err != nil {
		payload := map[string]any{
			"sid": randB64(24),
			"iat": time.Now().Unix(),
		}
		b, _ := json.Marshal(payload)
		p := base64.RawURLEncoding.EncodeToString(b)
		sig := signHMAC(s.cfg.Security.ActionSecret, p)
		token := p + "." + sig
		http.SetCookie(w, &http.Cookie{
			Name:     actCookie,
			Value:    token,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Secure:   secure,
			HttpOnly: true,
		})
	}
}

func (s *Server) allowCSRF(r *http.Request) bool {
	c, err := r.Cookie("__vlt_csrf")
	if err != nil || c.Value == "" {
		return false
	}
	h := r.Header.Get("X-Vlt-Csrf")
	if h == "" {
		h = r.Header.Get("x-vlt-csrf")
	}
	return h != "" && h == c.Value
}

func (s *Server) verifyActCookie(r *http.Request) (sid string, ok bool) {
	c, err := r.Cookie("__vlt_act")
	if err != nil || c.Value == "" {
		return "", false
	}
	parts := strings.Split(c.Value, ".")
	if len(parts) != 2 {
		return "", false
	}
	payloadB64 := parts[0]
	sig := parts[1]
	exp := signHMAC(s.cfg.Security.ActionSecret, payloadB64)
	if !hmac.Equal([]byte(sig), []byte(exp)) {
		return "", false
	}
	raw, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return "", false
	}
	var p struct {
		SID string `json:"sid"`
		IAT int64  `json:"iat"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return "", false
	}
	if p.SID == "" || p.IAT == 0 {
		return "", false
	}
	if time.Since(time.Unix(p.IAT, 0)) > 7*24*time.Hour {
		return "", false
	}
	return p.SID, true
}

func (s *Server) allowFresh(ts int64) bool {
	now := time.Now().UnixMilli()
	d := now - ts
	if d < 0 {
		d = -d
	}
	return d <= int64(2*time.Minute/time.Millisecond)
}

func (s *Server) allowReplay(sid, action, nonce string) bool {
	now := time.Now()
	k := sid + ":" + action + ":" + nonce

	s.replayMu.Lock()
	defer s.replayMu.Unlock()

	if exp, ok := s.replay[k]; ok && exp.After(now) {
		return false
	}
	if len(s.replay) > 20000 {
		for kk, exp := range s.replay {
			if !exp.After(now) {
				delete(s.replay, kk)
			}
		}
	}
	s.replay[k] = now.Add(5 * time.Minute)
	return true
}

func randB64(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func signHMAC(secret, msg string) string {
	h := hmac.New(sha256.New, []byte(secret))
	_, _ = h.Write([]byte(msg))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
