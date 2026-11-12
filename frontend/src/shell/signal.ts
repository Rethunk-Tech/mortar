/** A set of listeners that `emit` calls in subscription order. `subscribe` returns the unsubscribe function. */
export function createSignal() {
  const listeners = new Set<() => void>()
  return {
    emit() {
      for (const listener of listeners) {
        listener()
      }
    },
    subscribe(listener: () => void): () => void {
      listeners.add(listener)
      return () => {
        listeners.delete(listener)
      }
    },
  }
}
