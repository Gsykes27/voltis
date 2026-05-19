export async function callAction<T = unknown>(name: string, data: any): Promise<T> {
  const csrf = getCookie("__vlt_csrf")
  const res = await fetch("/__vlt/actions/" + encodeURIComponent(name), {
    method: "POST",
    headers: { "content-type": "application/json", "x-vlt-csrf": csrf },
    body: JSON.stringify({ data, ts: Date.now(), nonce: nonceHex() }),
    credentials: "include",
  })
  if (!res.ok) throw new Error(await res.text())
  const json = await res.json()
  return json.result as T
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
    try {
      msg = JSON.parse(ev.data)
    } catch {
      return
    }
    if (!msg) return
    if (msg.t === "event" && msg.ch === ch) onEvent(msg.data)
  })

  return {
    publish: (data: any) => send({ t: "pub", ch, data }),
    close: () => ws.close(),
  }
}
