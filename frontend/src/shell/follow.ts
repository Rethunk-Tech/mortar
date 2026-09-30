import { Events } from '@wailsio/runtime'

type Data<E extends keyof Events.CustomEvents> = Events.CustomEvents[E]

type Subscribe<E extends keyof Events.CustomEvents> = (
  name: E,
  cb: (event: { data: Data<E> }) => void,
) => void

// Keeps a store on an evented value: subscribes, then fetches the current one.
// A fetch that finishes after any event is dropped so a slower snapshot cannot rewind live state.
export async function follow<E extends keyof Events.CustomEvents>(
  name: E,
  fetch: () => Promise<Data<E>>,
  apply: (next: Data<E>, first: boolean) => void,
  subscribe: Subscribe<E> = (n, cb) => {
    Events.On(n, cb)
  },
): Promise<void> {
  let seen = false
  subscribe(name, (event) => {
    apply(event.data, !seen)
    seen = true
  })
  const current = await fetch()
  if (!seen) {
    apply(current, true)
    seen = true
  }
}
