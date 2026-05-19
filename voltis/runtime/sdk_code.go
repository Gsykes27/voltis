package runtime

const sdkJS = `
function __vlt_cookie(name) {
  const m = document.cookie.match(new RegExp("(^|;\\s*)" + name.replace(/[.*+?^${}()|[\\]\\\\]/g, "\\\\$&") + "=([^;]*)"))
  return m ? decodeURIComponent(m[2]) : ""
}

function __vlt_nonce() {
  const b = new Uint8Array(16)
  crypto.getRandomValues(b)
  let s = ""
  for (let i = 0; i < b.length; i++) s += b[i].toString(16).padStart(2, "0")
  return s
}

export function callAction(name, data) {
  const csrf = __vlt_cookie("__vlt_csrf")
  return fetch("/__vlt/actions/" + encodeURIComponent(name), {
    method: "POST",
    headers: { "content-type": "application/json", "x-vlt-csrf": csrf },
    body: JSON.stringify({ data, ts: Date.now(), nonce: __vlt_nonce() }),
    credentials: "include",
  }).then(async (res) => {
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  }).then((j) => j.result)
}

export function connectChannel(ch, onEvent) {
  const proto = location.protocol === "https:" ? "wss:" : "ws:"
  const ws = new WebSocket(proto + "//" + location.host + "/__vlt/ws")
  const send = (obj) => ws.readyState === 1 ? ws.send(JSON.stringify(obj)) : null

  ws.addEventListener("open", () => send({ t: "sub", ch }))
  ws.addEventListener("message", (ev) => {
    let msg = null
    try { msg = JSON.parse(ev.data) } catch { return }
    if (!msg) return
    if (msg.t === "event" && msg.ch === ch) onEvent(msg.data)
  })

  return {
    publish: (data) => send({ t: "pub", ch, data }),
    close: () => ws.close(),
  }
}
`
