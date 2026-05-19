import React from "react"
import { createBrowserRouter } from "react-router-dom"

type Mod = { default: React.ComponentType }

const modules = import.meta.glob("./routes/**/*.tsx", { eager: true }) as Record<string, Mod>

function fileToPath(file: string): string {
  const rel = file.replace("./routes/", "").replace(/\.tsx$/, "")
  if (rel === "index") return "/"
  const parts = rel.split("/")
  const mapped = parts
    .map((p) => {
      if (p.startsWith("[") && p.endsWith("]")) return ":" + p.slice(1, -1)
      if (p === "index") return ""
      return p
    })
    .filter(Boolean)
  return "/" + mapped.join("/")
}

const routes = Object.entries(modules).map(([file, mod]) => ({
  path: fileToPath(file),
  element: React.createElement(mod.default),
}))

export const router = createBrowserRouter(routes)

