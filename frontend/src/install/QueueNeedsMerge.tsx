import type { Item } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/models.ts'
import { MergeCallout } from './MergeCallout.tsx'

export function QueueNeedsMerge({ items }: { items: Item[] }) {
  const waiting = items.filter((i) => i.state === 'needs-merge')
  return (
    <>
      {waiting.map((item) => (
        <MergeCallout key={item.id} item={item} />
      ))}
    </>
  )
}
