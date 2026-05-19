# Voltis Architecture (Vite + React + Go Runtime)

Voltis é um meta-framework fullstack realtime-first cujo runtime principal é Go. O frontend é um app Vite + React (CSR), enquanto o backend (Go) fornece:

- Server Actions (endpoints HTTP)
- Websocket channels (pub/sub)
- Proxies de dev para Vite (HMR do Vite intacto)
- Servir assets estáticos do build do Vite em produção

## Estrutura (atual)

- `cmd/voltis` — CLI entrypoint
- `voltis/cli` — `create`, `dev`, `build`, `start`
- `voltis/builder` — build via `npm run build`
- `voltis/runtime` — HTTP runtime + websocket + actions + dev proxy
- `examples/basic` — app Vite+React consumindo `/__vlt/*`

## Runtime HTTP (Go)

Rotas internas:

- `GET /__vlt/sdk.js` — SDK ESM para o client (fetch actions + ws channel)
- `POST /__vlt/actions/:name` — Server Actions (executadas em Go)
- `GET /__vlt/ws` — websocket (sub/unsub/pub/event)
- `GET /public/*` — arquivos estáticos do app

Modo dev:

- `voltis dev` inicia o Vite e faz proxy de todo o restante (`/`, `/@vite/*`, `/src/*`, `/assets/*`) para o Vite.

Modo prod:

- `voltis build` roda `npm run build` do app (Vite) e gera `dist/client`
- `voltis start` serve `dist/client/*` e faz fallback de SPA para `dist/client/index.html`

## Realtime / Websocket

Protocolo JSON:

- `{"t":"sub","ch":"counter"}` — subscribe
- `{"t":"unsub","ch":"counter"}` — unsubscribe
- `{"t":"pub","ch":"counter","data":{...}}` — publish
- `{"t":"event","ch":"counter","data":{...}}` — server -> client broadcast

## Server Actions (Go)

No MVP atual, existe uma action de exemplo:

- `IncrementCounter` — incrementa um contador em memória e publica update no canal `counter`.

Próximos passos:

- Registro de actions por app (compilação do app em binário / plugin / WASM)
- Assinatura, CSRF, replay protection, rate limit distribuído (Redis)

