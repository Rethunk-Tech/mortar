import { Events } from '@wailsio/runtime'

type Data<E extends keyof Events.CustomEvents> = Events.CustomEvents[E]

// Keeps a store on an evented value: subscribes, then fetches the current one. Sequence numbers let a fetch that
// finishes after an event still apply; first marks the first value applied.
export async function follow<E extends keyof Events.CustomEvents>(
  name: E,
  fetch: () => Promise<Data<E>>,
  apply: (next: Data<E>, first: boolean) => void,
): Promise<void> {
  let seq = 0
  let applied = 0
  let seen = false
  Events.On(name, (event) => {
    seq += 1
    const n = seq
    apply(event.data, !seen)
    seen = true
    applied = n
  })
  const current = await fetch()
  seq += 1
  const n = seq
  if (n > applied) {
    apply(current, !seen)
    seen = true
  }
}
