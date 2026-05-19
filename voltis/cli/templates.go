package cli

const tmplAppPackageJSON = `{
  "name": "voltis-app",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "react": "^18.3.1",
    "react-dom": "^18.3.1",
    "react-router-dom": "^6.26.2"
  },
  "devDependencies": {
    "@types/react": "^18.3.12",
    "@types/react-dom": "^18.3.1",
    "@vitejs/plugin-react": "^4.3.1",
    "typescript": "^5.5.4",
    "vite": "^5.4.10"
  }
}
`

const tmplViteConfig = `import { defineConfig } from "vite"
import react from "@vitejs/plugin-react"

export default defineConfig({
  plugins: [react()],
  base: "/",
  build: {
    outDir: "../dist/client",
    emptyOutDir: true
  }
})
`

const tmplIndexHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width,initial-scale=1" />
    <title>Voltis</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
`

const tmplMainTSX = `import React from "react"
import ReactDOM from "react-dom/client"
import { RouterProvider } from "react-router-dom"
import { router } from "./router"
import "./styles.css"

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <RouterProvider router={router} />
  </React.StrictMode>
)
`

const tmplRouterTSX = `import React from "react"
import { createBrowserRouter } from "react-router-dom"

type Mod = { default: React.ComponentType }

const modules = import.meta.glob("./routes/**/*.tsx", { eager: true }) as Record<string, Mod>

function fileToPath(file: string): string {
  const rel = file.replace("./routes/", "").replace(/\\.tsx$/, "")
  if (rel === "index") return "/"
  const parts = rel.split("/")
  const mapped = parts.map((p) => {
    if (p.startsWith("[") && p.endsWith("]")) return ":" + p.slice(1, -1)
    if (p === "index") return ""
    return p
  }).filter(Boolean)
  return "/" + mapped.join("/")
}

const routes = Object.entries(modules).map(([file, mod]) => ({
  path: fileToPath(file),
  element: React.createElement(mod.default),
}))

export const router = createBrowserRouter(routes)
`

const tmplHomeRoute = `import React, { useEffect, useMemo, useState } from "react"
import { callAction, connectChannel } from "../voltis"

export default function Home() {
  const [counter, setCounter] = useState(0)
  const ch = useMemo(() => connectChannel("counter", (ev) => {
    if (ev && typeof ev.value === "number") setCounter(ev.value)
  }), [])

  useEffect(() => () => ch.close(), [ch])

  const onInc = async () => {
    const v = await callAction("IncrementCounter", {})
    setCounter(v)
  }

  return (
    <div className="wrap">
      <h1>Hello Voltis</h1>
      <p>Counter: {counter}</p>
      <button onClick={onInc}>Increment</button>
    </div>
  )
}
`

const tmplVoltisTS = `export async function callAction(name: string, data: any) {
  const csrf = getCookie("__vlt_csrf")
  const res = await fetch("/__vlt/actions/" + encodeURIComponent(name), {
    method: "POST",
    headers: { "content-type": "application/json", "x-vlt-csrf": csrf },
    body: JSON.stringify({ data, ts: Date.now(), nonce: nonceHex() }),
    credentials: "include"
  })
  if (!res.ok) throw new Error(await res.text())
  const json = await res.json()
  return json.result
}

function getCookie(name: string): string {
  const m = document.cookie.match(
    new RegExp("(^|;\\s*)" + name.replace(/[.*+?^${}()|[\\]\\\\]/g, "\\\\$&") + "=([^;]*)")
  )
  return m ? decodeURIComponent(m[2]) : ""
}

function nonceHex(): string {
  const b = new Uint8Array(16)
  crypto.getRandomValues(b)
  let s = ""
  for (let i = 0; i < b.length; i++) s += b[i].toString(16).padStart(2, "0")
  return s
}

export function connectChannel(ch: string, onEvent: (data: any) => void) {
  const proto = location.protocol === "https:" ? "wss:" : "ws:"
  const ws = new WebSocket(proto + "//" + location.host + "/__vlt/ws")
  const send = (obj: any) => (ws.readyState === 1 ? ws.send(JSON.stringify(obj)) : null)

  ws.addEventListener("open", () => send({ t: "sub", ch }))
  ws.addEventListener("message", (ev) => {
    let msg: any = null
    try { msg = JSON.parse(ev.data) } catch { return }
    if (!msg) return
    if (msg.t === "event" && msg.ch === ch) onEvent(msg.data)
  })

  return {
    publish: (data: any) => send({ t: "pub", ch, data }),
    close: () => ws.close()
  }
}
`

const tmplTSConfig = `{
  "compilerOptions": {
    "target": "ES2022",
    "useDefineForClassFields": true,
    "lib": ["ES2022", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "Bundler",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true
  },
  "include": ["src"]
}
`

const tmplServerMainGo = `package main

import (
  "context"
  "flag"
  "os"
  "path/filepath"

  "example.com/voltis-app/server/actions"
  "github.com/voltis/voltis/voltis/runtime"
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
    Actions: actions.Registry,
  })
  if err != nil {
    panic(err)
  }

  if err := s.Serve(context.Background()); err != nil {
    panic(err)
  }
}
`

const tmplServerRegistryGo = `package actions

import "github.com/voltis/voltis/voltis/runtime"

var Registry = runtime.NewActionRegistry()
`

const tmplServerCounterActionGo = `package actions

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
`

const tmplStylesCSS = `:root { color-scheme: light; }
html, body { height: 100%; }
body { margin: 0; font-family: system-ui, -apple-system, Segoe UI, Roboto, sans-serif; background: #fff; color: #111; }
.wrap { padding: 24px; }
button { padding: 8px 12px; }
`

const tmplVoltisConfig = `{
  "appDir": "./app",
  "distDir": "./dist",
  "http": { "addr": ":3000" },
  "dev": { "vitePort": 5173 },
  "security": { "actionSecret": "dev-secret-change-me" }
}
`

const tmplGitIgnore = `/dist/
/coverage/
/.cache/
/.idea/
/.vscode/
/.DS_Store
Thumbs.db

*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out
*.prof
*.pprof
*.trace

go.work
go.work.sum
/vendor/

**/node_modules/
**/npm-debug.log*
**/yarn-debug.log*
**/yarn-error.log*
**/pnpm-debug.log*
**/.pnpm-store/
**/.npm/
**/.yarn/

**/dist/
**/.vite/

.env
.env.*
!.env.example

*.log
`
