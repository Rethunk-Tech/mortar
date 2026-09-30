import { Events } from '@wailsio/runtime'

type Data<E extends keyof Events.CustomEvents> = Events.CustomEvents[E]

// Keeps a store on an evented value: subscribes, then fetches the current one. Nothing orders the fetch against
// events, so once an event has landed the fetched value is older and dropped; first marks the first value applied.
export async function follow<E extends keyof Events.CustomEvents>(
  name: E,
  fetch: () => Promise<Data<E>>,
  apply: (next: Data<E>, first: boolean) => void,
): Promise<void> {
  let seen = false
  Events.On(name, (event) => {
    apply(event.data, !seen)
    seen = true
  })
  const current = await fetch()
  if (!seen) {
    seen = true
    apply(current, true)
  }
}
