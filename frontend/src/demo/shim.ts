import recorded from './generated/recording.json' with { type: 'json' }
import { callKey, type Recording } from './replay.ts'

const recording: Recording = recorded
const JSON_TYPE = { 'Content-Type': 'application/json' }
const pageFetch = globalThis.fetch.bind(globalThis)

// A call the walk did not record (a view outside the demo, a change the visitor asks for) gets null, which every
// binding reads as empty; nothing reaches a backend, so a change is a no-op.
function answer(body: string): Response {
  const reply = recording[callKey(body)]
  if (!reply) {
    return new Response('null', { headers: JSON_TYPE })
  }
  return 'json' in reply
    ? new Response(JSON.stringify(reply.json), { status: reply.status, headers: JSON_TYPE })
    : new Response(reply.text, { status: reply.status })
}

// The runtime posts every backend call to /wails/runtime and probes /wails/custom.js, both answered here, so the demo
// asks its host for nothing but its own files.
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
