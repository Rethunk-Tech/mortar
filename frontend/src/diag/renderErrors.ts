import { LogRenderError } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/support/service.ts'

const MAX_STACKS = 20
const seen = new Set<string>()

interface Info {
  componentStack?: string | undefined
}

// React's own report still reaches the console through reportError, as React's default does; mortar.log gets one line per distinct error and component stack,
// twenty a session at most, since a render loop repeats the same stack thousands of times.
export function logRenderError(
  error: unknown,
  info: Info,
  send: (message: string, stack: string) => Promise<void> = LogRenderError,
): boolean {
  globalThis.reportError(error)
  const stack = info.componentStack ?? ''
  const message = error instanceof Error ? error.message : String(error)
  const key = `${message}\n${stack}`
  if (seen.has(key) || seen.size >= MAX_STACKS) {
    return false
  }
  seen.add(key)
  send(message, stack).catch(() => undefined)
  return true
}
