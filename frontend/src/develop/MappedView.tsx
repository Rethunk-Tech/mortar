import { Box } from '@mui/material'
import { useMemo, useState } from 'react'
import type { Mapped } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import { CodeView } from './CodeView.tsx'

const NONE: number[] = []

// The C# line picked, from either side; the other side scrolls to it.
interface Pick {
  cs: number
  from: 'cs' | 'il'
}

/** C# and IL side by side: clicking a line on one side tints it and its counterpart lines on the other. */
export function MappedView({ mapped, line }: { mapped: Mapped; line: number }) {
  const links = mapped.links ?? []
  const [pick, setPick] = useState<Pick | null>(null)
  const ilOf = useMemo(() => {
    const byCs = new Map<number, number[]>()
    links.forEach((from, il) => {
      if (from >= 0) {
        byCs.set(from, [...(byCs.get(from) ?? []), il])
      }
    })
    return byCs
  }, [links])
  const cs = pick?.cs ?? -1
  const ilLines = ilOf.get(cs) ?? NONE
  const csMarked = useMemo(() => new Set(cs >= 0 ? [cs] : []), [cs])
  const ilMarked = useMemo(() => new Set(ilLines), [ilLines])
  return (
    <Box sx={{ flex: 1, minHeight: 0, display: 'grid', gridTemplateColumns: '1fr 1fr' }}>
      <Box
        sx={{ display: 'flex', minWidth: 0, borderRight: '1px solid var(--mortar-hairline-muted)' }}
      >
        <CodeView
          source={mapped.csharp}
          il={false}
          line={line}
          follow={pick?.from === 'il' ? cs : -1}
          marked={csMarked}
          onPick={(n) => setPick({ cs: n, from: 'cs' })}
        />
      </Box>
      <Box sx={{ display: 'flex', minWidth: 0 }}>
        <CodeView
          source={mapped.il}
          il={true}
          line={-1}
          follow={pick?.from === 'cs' ? (ilLines[0] ?? -1) : -1}
          marked={ilMarked}
          onPick={(n) => setPick({ cs: links[n] ?? -1, from: 'il' })}
        />
      </Box>
    </Box>
  )
}
