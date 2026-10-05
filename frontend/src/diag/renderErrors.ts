import { LogRenderError } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/support/service.ts'

const MAX_STACKS = 20
const MIN_GAP_MS = 1000
const seen = new Set<string>()
let last = 0

interface Info {
  componentStack?: string | undefined
}

// One line in mortar.log per distinct component stack, at most one a second and twenty a session: a render loop
// repeats the same stack thousands of times.
export function logRenderError(error: unknown, info: Info, now = Date.now()): boolean {
  const stack = info.componentStack ?? ''
  const message = error instanceof Error ? error.message : String(error)
  const key = `${message}\n${stack}`
  if (seen.has(key) || seen.size >= MAX_STACKS || now - last < MIN_GAP_MS) {
    return false
  }
  seen.add(key)
  last = now
  LogRenderError(message, stack).catch(() => undefined)
  return true
}
