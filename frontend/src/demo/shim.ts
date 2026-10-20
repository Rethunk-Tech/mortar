import { Backend } from './backend.ts'
import recorded from './generated/recording.json' with { type: 'json' }
import { callKey, type Recording } from './replay.ts'

const recording: Recording = recorded
const backend = new Backend(recording)
const JSON_TYPE = { 'Content-Type': 'application/json' }
const pageFetch = globalThis.fetch.bind(globalThis)

function answer(body: string): Response {
  const reply = backend.answer(callKey(body))
  return 'json' in reply
    ? new Response(JSON.stringify(reply.json), { status: reply.status, headers: JSON_TYPE })
    : new Response(reply.text, { status: reply.status })
}

// The runtime posts every backend call to /wails/runtime and probes /wails/custom.js (the event socket, whose events
// the demo dispatches itself), both answered here, so the demo asks its host for nothing but its own files.
function demoFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const url = new URL(input instanceof Request ? input.url : input, location.href)
  if (url.origin !== location.origin || !url.pathname.startsWith('/wails/')) {
    return pageFetch(input, init)
  }
  const body = init?.body
  return Promise.resolve(
    url.pathname === '/wails/runtime' && typeof body === 'string'
      ? answer(body)
      : new Response(null, { status: 404 }),
  )
}

globalThis.fetch = Object.assign(demoFetch, { preconnect: globalThis.fetch.preconnect })
