import type { Item } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/models.ts'
import { NeedsRootCallout } from './NeedsRootCallout.tsx'

export function QueueNeedsRoot({ items }: { items: Item[] }) {
  const waiting = items.filter((i) => i.state === 'needs-root')
  return (
    <>
      {waiting.map((item) => (
        <NeedsRootCallout key={item.id} item={item} />
      ))}
    </>
  )
}
