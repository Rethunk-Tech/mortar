import { GlobalRegistrator } from '@happy-dom/global-registrator'

// One DOM for the whole run: a file that registered and unregistered its own left the testing library's cached
// `document` pointing at a closed window for every component test after it.
GlobalRegistrator.register({ url: 'http://localhost/' })
Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true })

// No Wails server runs under test: a binding a test did not mock (a store subscriber reacting to another file's state)
// answers null, as an empty Go result does, instead of a connection refused on port 80.
const realFetch = globalThis.fetch
// The runtime's optional /wails/custom.js answers empty for the same reason.
const answers: Record<string, () => Response> = {
  '/wails/runtime': () => Response.json(null),
  '/wails/custom.js': () => new Response(''),
}
globalThis.fetch = Object.assign(
  (input: Parameters<typeof fetch>[0], init?: Parameters<typeof fetch>[1]) => {
    const answer =
      answers[new URL(input instanceof Request ? input.url : input, location.href).pathname]
    return answer ? Promise.resolve(answer()) : realFetch(input, init)
  },
  realFetch,
)
