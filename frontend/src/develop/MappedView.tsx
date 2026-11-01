import { Box } from '@mui/material'
import { useMemo, useState } from 'react'
import type { Mapped } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import { CodeView } from './CodeView.tsx'
import { nearestLine } from './nearestLine.ts'
import type { SymbolClick } from './symbolAt.ts'

const NONE: number[] = []

// The C# line clicked and the nearest one with IL, from either side; the other side scrolls to it. Each click is a
// new Pick, so clicking the same line again scrolls back to it.
interface Pick {
  clicked: number
  cs: number
  from: 'cs' | 'il'
}

/** C# and IL side by side: clicking a line on one side tints it and its counterpart lines on the other. */
export function MappedView({
  mapped,
  line,
  onSymbol,
}: {
  mapped: Mapped
  line: number
  onSymbol?: ((at: SymbolClick) => void) | undefined
}) {
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
  // A C# click tints the line clicked as well as the one it resolved to.
  const csMarked = useMemo(() => {
    const lines = pick?.from === 'cs' ? [pick.clicked, cs] : [cs]
    return new Set(lines.filter((n) => n >= 0))
  }, [pick, cs])
  const ilMarked = useMemo(() => new Set(ilLines), [ilLines])
  const csCount = useMemo(() => mapped.csharp.split('\n').length, [mapped.csharp])
  const pickCs = (n: number) =>
    setPick({ clicked: n, cs: nearestLine(n, csCount, (c) => ilOf.has(c)), from: 'cs' })
  // An IL line with no C# of its own (a label, the locals) takes the C# of the nearest one that has.
  const pickIl = (n: number) => {
    const linked = nearestLine(n, links.length, (i) => (links[i] ?? -1) >= 0)
    setPick({ clicked: n, cs: linked >= 0 ? (links[linked] ?? -1) : -1, from: 'il' })
  }
  const csFollow = useMemo(() => (pick?.from === 'il' && cs >= 0 ? { line: cs } : null), [pick, cs])
  const ilFollow = useMemo(
    () => (pick?.from === 'cs' && ilLines[0] !== undefined ? { line: ilLines[0] } : null),
    [pick, ilLines],
  )
  return (
    <Box sx={{ flex: 1, minHeight: 0, display: 'grid', gridTemplateColumns: '1fr 1fr' }}>
      <Box
        sx={{ display: 'flex', minWidth: 0, borderRight: '1px solid var(--mortar-hairline-muted)' }}
      >
        <CodeView
          source={mapped.csharp}
          il={false}
          line={line}
          follow={csFollow}
          marked={csMarked}
          onPick={pickCs}
          onSymbol={onSymbol}
        />
      </Box>
      <Box sx={{ display: 'flex', minWidth: 0 }}>
        <CodeView
          source={mapped.il}
          il={true}
          line={-1}
          follow={ilFollow}
          marked={ilMarked}
          onPick={pickIl}
          onSymbol={onSymbol}
        />
      </Box>
    </Box>
  )
}
