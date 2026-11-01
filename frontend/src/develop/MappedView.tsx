import { Box } from '@mui/material'
import { useEffect, useMemo, useState } from 'react'
import type { Mapped } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import { CodeView } from './CodeView.tsx'
import { declarationLine, declaredIn } from './declaration.ts'
import { nearestLine } from './nearestLine.ts'
import type { SymbolClick } from './symbolAt.ts'

// The lines tinted on each side, and which side the pick came from: the other side scrolls to its first line (both
// for a member picked in the search). Each pick is a new object, so picking the same line again scrolls back to it.
interface Pick {
  cs: number[]
  il: number[]
  from: 'cs' | 'il' | 'search'
}

// The member the search picked: its declaration opens on both sides.
export interface PickedMember {
  type: string
  kind: string
  name: string
}

/**
 * C# and IL side by side: clicking a line on one side tints it and its counterpart on the other. A declaration's
 * counterpart is the member's declaration on the other side; a statement's is the IL compiled from it.
 */
export function MappedView({
  mapped,
  member,
  onSymbol,
}: {
  mapped: Mapped
  member: PickedMember
  onSymbol?: ((at: SymbolClick) => void) | undefined
}) {
  const links = mapped.links ?? []
  const csLines = useMemo(() => mapped.csharp.split('\n'), [mapped.csharp])
  const ilLines = useMemo(() => mapped.il.split('\n'), [mapped.il])
  const ilOf = useMemo(() => {
    const byCs = new Map<number, number[]>()
    links.forEach((from, il) => {
      if (from >= 0) {
        byCs.set(from, [...(byCs.get(from) ?? []), il])
      }
    })
    return byCs
  }, [links])
  const [pick, setPick] = useState<Pick | null>(null)
  useEffect(() => {
    const cs = declarationLine(mapped.csharp, member.type, member.kind, member.name)
    const il = declarationLine(mapped.il, member.type, member.kind, member.name)
    setPick({ cs: cs >= 0 ? [cs] : [], il: il >= 0 ? [il] : [], from: 'search' })
  }, [mapped, member.type, member.kind, member.name])
  const pickCs = (n: number) => {
    const decl = declaredIn(csLines[n] ?? '')
    const il = decl ? declarationLine(mapped.il, member.type, decl.kind, decl.name) : -1
    if (il >= 0) {
      setPick({ cs: [n], il: [il], from: 'cs' })
      return
    }
    // A line without IL of its own (a brace, a comment) takes the nearest one that has, tinted beside it.
    const cs = nearestLine(n, csLines.length, (c) => ilOf.has(c))
    setPick({ cs: cs >= 0 && cs !== n ? [n, cs] : [n], il: ilOf.get(cs) ?? [], from: 'cs' })
  }
  const pickIl = (n: number) => {
    const decl = declaredIn(ilLines[n] ?? '')
    const cs = decl ? declarationLine(mapped.csharp, member.type, decl.kind, decl.name) : -1
    if (cs >= 0) {
      setPick({ cs: [cs], il: [n], from: 'il' })
      return
    }
    // An IL line with no C# of its own (a label, the locals) takes the C# of the nearest one that has.
    const linked = nearestLine(n, links.length, (i) => (links[i] ?? -1) >= 0)
    const to = linked >= 0 ? (links[linked] ?? -1) : -1
    setPick({ cs: to >= 0 ? [to] : [], il: [n], from: 'il' })
  }
  const csMarked = useMemo(() => new Set(pick?.cs ?? []), [pick])
  const ilMarked = useMemo(() => new Set(pick?.il ?? []), [pick])
  const csFollow = useMemo(
    () => (pick && pick.from !== 'cs' && pick.cs[0] !== undefined ? { line: pick.cs[0] } : null),
    [pick],
  )
  const ilFollow = useMemo(
    () => (pick && pick.from !== 'il' && pick.il[0] !== undefined ? { line: pick.il[0] } : null),
    [pick],
  )
  return (
    <Box sx={{ flex: 1, minHeight: 0, display: 'grid', gridTemplateColumns: '1fr 1fr' }}>
      <Box
        sx={{ display: 'flex', minWidth: 0, borderRight: '1px solid var(--mortar-hairline-muted)' }}
      >
        <CodeView
          source={mapped.csharp}
          il={false}
          line={-1}
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
