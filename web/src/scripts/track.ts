// Counts an event in Plausible, whose script Base.astro loads. Props must never carry anything that identifies a person
// or a shared profile's contents.
type Plausible = (event: string, options?: { props?: Record<string, string> }) => void

export function track(event: string, props?: Record<string, string>) {
  ;(globalThis as { plausible?: Plausible }).plausible?.(event, props ? { props } : undefined)
}
