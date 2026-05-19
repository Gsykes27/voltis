import React, { useEffect, useMemo, useRef, useState } from "react"
import { callAction, connectChannel } from "../voltis"

type Ticket = {
  id: number
  title: string
  priority: string
  status: "open" | "resolved"
  createdAt: number
  updatedAt: number
}

type ChatMessage = {
  id: number
  room: string
  author: string
  text: string
  createdAt: number
}

function formatTime(ms: number): string {
  try {
    return new Date(ms).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
  } catch {
    return ""
  }
}

function CodeBlock(props: { title?: string; code: string }) {
  return (
    <div className="codeBlock" aria-label={props.title || "Code"}>
      <pre>
        <code>{props.code}</code>
      </pre>
    </div>
  )
}

function CounterDemo() {
  const [counter, setCounter] = useState(0)
  const ch = useMemo(
    () =>
      connectChannel("counter", (ev) => {
        if (ev && typeof ev.value === "number") setCounter(ev.value)
      }),
    []
  )

  useEffect(() => {
    callAction<number>("GetCounter", {}).then(setCounter).catch(() => {})
    return () => ch.close()
  }, [ch])

  const onInc = async () => {
    const v = await callAction<number>("IncrementCounter", {})
    setCounter(v)
  }

  return (
    <div className="demoCard" id="demo-counter">
      <div className="row">
        <div>
          <div className="cardTitle">Counter (Realtime)</div>
          <div className="cardP">Server action + broadcast no canal counter.</div>
        </div>
        <span className="pill" aria-label="Counter value">
          {counter}
        </span>
      </div>
      <div className="ctaRow" style={{ marginTop: 10 }}>
        <button className="btn btnPrimary" onClick={onInc}>
          Incrementar
        </button>
        <button className="btn" onClick={() => callAction<number>("GetCounter", {}).then(setCounter)}>
          Recarregar
        </button>
      </div>
      <div style={{ marginTop: 12 }}>
        <CodeBlock
          title="Counter code"
          code={`import { callAction, connectChannel } from "../voltis"

const ch = connectChannel("counter", (ev) => setCounter(ev.value))
await callAction("IncrementCounter", {})`}
        />
      </div>
    </div>
  )
}

function TicketsDemo() {
  const [title, setTitle] = useState("")
  const [priority, setPriority] = useState("normal")
  const [tickets, setTickets] = useState<Ticket[]>([])
  const [loading, setLoading] = useState(false)

  const refresh = async () => {
    const list = await callAction<Ticket[]>("ListTickets", {})
    setTickets(list)
  }

  const ch = useMemo(
    () =>
      connectChannel("tickets", () => {
        refresh().catch(() => {})
      }),
    []
  )

  useEffect(() => {
    refresh().catch(() => {})
    return () => ch.close()
  }, [ch])

  const create = async () => {
    setLoading(true)
    try {
      await callAction("CreateTicket", { title, priority })
      setTitle("")
      await refresh()
    } finally {
      setLoading(false)
    }
  }

  const resolve = async (id: number) => {
    await callAction("ResolveTicket", { id })
    await refresh()
  }

  return (
    <div className="demoCard" id="demo-tickets">
      <div className="row">
        <div>
          <div className="cardTitle">Tickets (Omnichannel)</div>
          <div className="cardP">Criação, listagem, resolução e updates via websocket.</div>
        </div>
        <span className="pill">{tickets.length} itens</span>
      </div>

      <div className="inputRow" aria-label="Create ticket">
        <label style={{ display: "contents" }}>
          <span className="pill" style={{ borderStyle: "dashed" }}>
            Título
          </span>
          <input
            className="input"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="Ex: Usuário não consegue logar"
            aria-label="Título do ticket"
          />
        </label>
        <label style={{ display: "contents" }}>
          <span className="pill" style={{ borderStyle: "dashed" }}>
            Prioridade
          </span>
          <select className="select" value={priority} onChange={(e) => setPriority(e.target.value)} aria-label="Prioridade">
            <option value="low">low</option>
            <option value="normal">normal</option>
            <option value="high">high</option>
          </select>
        </label>
        <button className="btn btnPrimary" disabled={loading || title.trim() === ""} onClick={create} aria-disabled={loading}>
          Criar ticket
        </button>
      </div>

      <div className="list" role="list" aria-label="Ticket list">
        {tickets.map((t) => (
          <div className="item" key={t.id} role="listitem">
            <div className="itemTop">
              <div className="itemTitle">{t.title}</div>
              <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
                <span className={`tag tagPrio`} aria-label={`priority ${t.priority}`}>
                  {t.priority}
                </span>
                <span className={`tag ${t.status === "resolved" ? "tagResolved" : "tagOpen"}`}>{t.status}</span>
                <button className="btn" onClick={() => resolve(t.id)} disabled={t.status === "resolved"}>
                  Resolver
                </button>
              </div>
            </div>
            <div className="itemMeta">
              #{t.id} • criado {formatTime(t.createdAt)} • atualizado {formatTime(t.updatedAt)}
            </div>
          </div>
        ))}
        {tickets.length === 0 ? <div className="cardP">Nenhum ticket ainda. Crie um acima.</div> : null}
      </div>

      <div style={{ marginTop: 12 }}>
        <CodeBlock
          title="Ticket actions"
          code={`await callAction("CreateTicket", { title, priority })
const tickets = await callAction("ListTickets", {})
await callAction("ResolveTicket", { id })

connectChannel("tickets", (ev) => {
  // ev.type: created | resolved
  // ev.ticket: ticket
})`}
        />
      </div>
    </div>
  )
}

function ChatDemo() {
  const room = "support"
  const [author, setAuthor] = useState("Agent")
  const [text, setText] = useState("")
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const listRef = useRef<HTMLDivElement | null>(null)

  const refresh = async () => {
    const list = await callAction<ChatMessage[]>("ListChatMessages", { room })
    setMessages(list)
  }

  const ch = useMemo(
    () =>
      connectChannel("chat:" + room, (ev) => {
        if (ev && ev.type === "message" && ev.message) {
          setMessages((m) => [...m, ev.message as ChatMessage])
        }
      }),
    [room]
  )

  useEffect(() => {
    refresh().catch(() => {})
    return () => ch.close()
  }, [ch])

  useEffect(() => {
    listRef.current?.scrollTo(0, listRef.current.scrollHeight)
  }, [messages.length])

  const send = async () => {
    const trimmed = text.trim()
    if (!trimmed) return
    setText("")
    await callAction("SendChatMessage", { room, author, text: trimmed })
  }

  return (
    <div className="demoCard" id="demo-chat">
      <div className="row">
        <div>
          <div className="cardTitle">Chat (Realtime)</div>
          <div className="cardP">Mensagens em room via websocket + action de envio.</div>
        </div>
        <span className="pill">room: {room}</span>
      </div>

      <div className="inputRow" style={{ marginTop: 10 }}>
        <label style={{ display: "contents" }}>
          <span className="pill" style={{ borderStyle: "dashed" }}>
            Autor
          </span>
          <input className="input" value={author} onChange={(e) => setAuthor(e.target.value)} aria-label="Autor" />
        </label>
        <label style={{ display: "contents" }}>
          <span className="pill" style={{ borderStyle: "dashed" }}>
            Mensagem
          </span>
          <input
            className="input"
            value={text}
            onChange={(e) => setText(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") send().catch(() => {})
            }}
            placeholder="Ex: Olá, como posso ajudar?"
            aria-label="Mensagem"
          />
        </label>
        <button className="btn btnPrimary" onClick={() => send().catch(() => {})}>
          Enviar
        </button>
      </div>

      <div
        ref={listRef}
        className="codeBlock"
        style={{ marginTop: 10, maxHeight: 220 }}
        role="log"
        aria-label="Chat log"
        aria-live="polite"
      >
        <pre style={{ margin: 0 }}>
          <code>
            {messages.length === 0
              ? "Sem mensagens ainda.\n"
              : messages.map((m) => `[${formatTime(m.createdAt)}] ${m.author}: ${m.text}`).join("\n")}
          </code>
        </pre>
      </div>

      <div style={{ marginTop: 12 }}>
        <CodeBlock
          title="Chat actions"
          code={`await callAction("SendChatMessage", { room: "support", author, text })
const history = await callAction("ListChatMessages", { room: "support" })
connectChannel("chat:support", (ev) => console.log(ev.message))`}
        />
      </div>
    </div>
  )
}

export default function Home() {
  const [active, setActive] = useState<"overview" | "docs" | "demos">("overview")

  useEffect(() => {
    const onHash = () => {
      const h = window.location.hash.replace("#", "")
      if (h === "docs") setActive("docs")
      else if (h === "demos") setActive("demos")
      else setActive("overview")
    }
    onHash()
    window.addEventListener("hashchange", onHash)
    return () => window.removeEventListener("hashchange", onHash)
  }, [])

  const curlExample = `# 1) pegar cookies (csrf + sessão)
curl -i http://localhost:3012/ -c cookies.txt > NUL

# 2) extrair csrf e chamar action
CSRF=$(cat cookies.txt | grep __vlt_csrf | awk '{print $7}')
curl -X POST http://localhost:3012/__vlt/actions/CreateTicket \\
  -b cookies.txt \\
  -H "content-type: application/json" \\
  -H "x-vlt-csrf: $CSRF" \\
  -d '{"data":{"title":"Bug no login","priority":"high"},"ts":1710000000000,"nonce":"<random>"}'`

  const goActionsExample = `switch name {
case "CreateTicket":
  title := data["title"].(string)
  t := db.TicketCreate(title, "high")
  ws.Publish(ctx, "tickets", map[string]any{"type":"created","ticket":t})
  return t, nil
case "SendChatMessage":
  msg := db.ChatAppend("support", "Agent", "Olá!")
  ws.Publish(ctx, "chat:support", map[string]any{"type":"message","message":msg})
  return msg, nil
}`

  return (
    <div className="page">
      <header className="topbar">
        <div className="topbarInner">
          <a className="brand" href="#overview" aria-label="Voltis home">
            <span className="brandMark" aria-hidden="true" />
            <span>Voltis</span>
          </a>
          <nav className="nav" aria-label="Primary">
            <a href="#overview" aria-current={active === "overview" ? "page" : undefined}>
              Intuito
            </a>
            <a href="#docs" aria-current={active === "docs" ? "page" : undefined}>
              Docs
            </a>
            <a href="#demos" aria-current={active === "demos" ? "page" : undefined}>
              Exemplos
            </a>
          </nav>
          <div style={{ display: "flex", gap: 10 }}>
            <a className="btn" href="http://localhost:3012/__vlt/sdk.js" target="_blank" rel="noreferrer">
              SDK
            </a>
            <a className="btn btnPrimary" href="#demos">
              Rodar demos
            </a>
          </div>
        </div>
      </header>

      <main className="container">
        <section className="hero" id="overview" aria-label="Hero">
          <div className="heroLeft">
            <div className="badgeRow">
              <span className="badge">Go Runtime</span>
              <span className="badge">Realtime-first</span>
              <span className="badge">Websocket-heavy</span>
              <span className="badge">Vite + React</span>
              <span className="badge">Cloud-native</span>
            </div>
            <h1 className="heroTitle">
              <span className="gradientText">Voltis</span> é um framework fullstack realtime-first,
              <br />
              com o backend em <span className="gradientText">Go</span>.
            </h1>
            <p className="heroSub">
              Voltis existe para apps em tempo real: chats, dashboards, omnichannel, colaboração e workloads com websocket. O frontend é um
              Vite + React (rápido de evoluir), enquanto server actions, filas, auth e realtime ficam do lado do Go.
            </p>
            <div className="ctaRow">
              <a className="btn btnPrimary" href="#docs">
                Ler docs rápidas
              </a>
              <a className="btn" href="#demos">
                Ver exemplos reais
              </a>
              <a className="btn" href="https://localhost" onClick={(e) => e.preventDefault()}>
                Deploy binário-first
              </a>
            </div>
          </div>

          <aside className="heroRight" aria-label="Quick facts">
            <div className="statGrid">
              <div className="stat">
                <div className="statK">Go</div>
                <div className="statV">Ações server-side, websocket e segurança no runtime.</div>
              </div>
              <div className="stat">
                <div className="statK">Vite</div>
                <div className="statV">HMR e toolchain moderna sem lock-in de framework.</div>
              </div>
              <div className="stat">
                <div className="statK">Realtime</div>
                <div className="statV">Pub/Sub com rooms e eventos distribuíveis (Redis/NATS).</div>
              </div>
              <div className="stat">
                <div className="statK">DX</div>
                <div className="statV">Logs limpos, dev proxy, e exemplos práticos embutidos.</div>
              </div>
            </div>

            <div className="card" style={{ marginTop: 6 }}>
              <div className="cardTitle">Princípio</div>
              <p className="cardP">
                O React renderiza e interage. O Go valida, executa e escala. Nada de SSR React no backend — performance e controle no runtime.
              </p>
            </div>
          </aside>
        </section>

        <section className="section" aria-label="What you get">
          <h2 className="sectionTitle">Por que Voltis?</h2>
          <div className="grid3">
            <div className="card">
              <p className="cardTitle">Server Actions em Go</p>
              <p className="cardP">Sem RPC “mágico”: HTTP seguro, com CSRF, nonce e replay protection (MVP).</p>
            </div>
            <div className="card">
              <p className="cardTitle">Websocket nativo</p>
              <p className="cardP">Canais/rooms com subscribe/publish e base pronta para Redis/NATS.</p>
            </div>
            <div className="card">
              <p className="cardTitle">Build Vite</p>
              <p className="cardP">Prod serve dist/client; dev usa proxy e mantém HMR do Vite.</p>
            </div>
          </div>
        </section>

        <section className="section" id="docs" aria-label="Docs">
          <h2 className="sectionTitle">Docs rápidas (embutidas)</h2>
          <div className="docGrid">
            <div className="docCard">
              <p className="cardTitle">Comandos</p>
              <p>
                Dev (Go inicia o Vite e faz proxy):
              </p>
              <CodeBlock
                title="Commands"
                code={`cd examples/basic/app
npm install

cd ..
go run ..\\..\\cmd\\voltis dev -addr :3012`}
              />
              <p>
                Produção local:
              </p>
              <CodeBlock title="Build+start" code={`go run ..\\..\\cmd\\voltis build\n\ngo run ..\\..\\cmd\\voltis start -addr :3013`} />
            </div>
            <div className="docCard">
              <p className="cardTitle">Modelo mental</p>
              <p>Voltis separa com clareza:</p>
              <ul>
                <li>UI e navegação no client (React Router).</li>
                <li>Server Actions no Go (validação e segurança).</li>
                <li>Realtime no Go (websocket + fanout).</li>
              </ul>
              <p>
                Client usa helpers em <span className="pill">src/voltis.ts</span> (ou o SDK em <span className="pill">/__vlt/sdk.js</span>).
              </p>
            </div>
          </div>

          <div className="section" style={{ marginTop: 14 }}>
            <h3 className="sectionTitle">Exemplos reais (Server Actions)</h3>
            <div className="demoGrid">
              <div className="demoCard">
                <div className="row">
                  <div>
                    <div className="cardTitle">No frontend (React)</div>
                    <div className="cardP">Call + realtime subscribe.</div>
                  </div>
                  <span className="pill">Type-safe friendly</span>
                </div>
                <div style={{ marginTop: 10 }}>
                  <CodeBlock
                    title="Frontend"
                    code={`import { callAction, connectChannel } from "../voltis"

connectChannel("tickets", (ev) => console.log(ev))
const t = await callAction("CreateTicket", { title: "Bug no login", priority: "high" })`}
                  />
                </div>
              </div>
              <div className="demoCard">
                <div className="row">
                  <div>
                    <div className="cardTitle">No backend (Go)</div>
                    <div className="cardP">Executa e publica eventos.</div>
                  </div>
                  <span className="pill">Goroutines</span>
                </div>
                <div style={{ marginTop: 10 }}>
                  <CodeBlock title="Backend" code={goActionsExample} />
                </div>
              </div>
            </div>

            <div className="section" style={{ marginTop: 14 }}>
              <h3 className="sectionTitle">Uso via HTTP (debug)</h3>
              <CodeBlock title="curl" code={curlExample} />
            </div>
          </div>
        </section>

        <section className="section" id="demos" aria-label="Demos">
          <h2 className="sectionTitle">Demos ao vivo</h2>
          <div className="demoGrid">
            <CounterDemo />
            <TicketsDemo />
          </div>
          <div className="section">
            <ChatDemo />
          </div>
        </section>

        <footer className="footer" aria-label="Footer">
          <div>
            <span className="pill">Voltis</span> • realtime-first • Go runtime • Vite+React
          </div>
          <div style={{ display: "flex", gap: 10 }}>
            <a className="btn" href="#overview">
              Topo
            </a>
            <a className="btn" href="#docs">
              Docs
            </a>
            <a className="btn btnPrimary" href="#demos">
              Exemplos
            </a>
          </div>
        </footer>
      </main>
    </div>
  )
}
