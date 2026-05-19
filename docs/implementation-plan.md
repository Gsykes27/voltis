# Voltis Implementation Plan (Revised: Vite+React)

## 1) Dev Experience

- Integrar `voltis dev` com:
  - detecção de porta livre do Vite
  - health-check + logs unificados
  - overlays de erro via Vite (já vem pronto)

## 2) Server Actions “de verdade” (Go)

- Registry de actions por app:
  - `app/server/actions.go` compilado no binário final do app
  - assinatura e ABI estável
- Segurança:
  - CSRF (cookie-based)
  - payload assinado + nonce + timestamp (replay protection)
  - rate limit por IP/usuário/action (Redis)
- Streaming responses:
  - SSE / chunked JSON para ações longas

## 3) Realtime Engine Distribuído

- Adapters:
  - Redis pub/sub (baixa latência)
  - NATS (fanout + request/reply; JetStream para durabilidade)
- Reconciliação de estado:
  - LWW e opção CRDT por tipo

## 4) Router

- Hoje: React Router no cliente + fallback SPA no Go
- Próximo: file-router com geração (Vite plugin) baseado em `app/src/routes`

## 5) SSR (sem React SSR)

Se for necessário “SSR” sem render React no backend:

- Streaming de shell HTML/headers no Go
- Prefetch de dados via server actions (hydrate client state)
- Edge: runtime Go minimal + cache agressivo

