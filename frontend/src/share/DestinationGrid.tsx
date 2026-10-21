import { useLingui } from '@lingui/react/macro'
import { List, Wifi } from 'lucide-react'
import type { ReactNode } from 'react'
import { Logo } from '../brand/Logo.tsx'
import { SourceLogo } from '../brand/sources/SourceLogo.tsx'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import type { Destination, DestinationEntry } from './methods.ts'
import { TileGrid } from './TileGrid.tsx'

const ICON = 32
const NEXUS = 'nexus'
const THUNDERSTORE = 'thunderstore'

export function DestinationGrid({
  entries,
  lastUsed,
  onChoose,
}: {
  entries: DestinationEntry[]
  lastUsed: Destination | null
  onChoose: (id: Destination) => void
}) {
  const { t } = useLingui()
  const tiles: Record<Destination, { icon: ReactNode; name: string; hint: string }> = {
    mortar: {
      icon: <Logo size={ICON} />,
      name: t`Mortar`,
      hint: t`A link or .mortar file`,
    },
    nexus: {
      icon: <SourceLogo id={NEXUS} size={ICON} />,
      name: t`Nexus Mods`,
      hint: t`A collection draft`,
    },
    thunderstore: {
      icon: <SourceLogo id={THUNDERSTORE} size={ICON} />,
      name: sourceLabel('thunderstore'),
      hint: t`An r2modman code or modpack`,
    },
    nearby: {
      icon: <Wifi size={ICON} />,
      name: t`Nearby computer`,
      hint: t`Send over your network`,
    },
    list: {
      icon: <List size={ICON} />,
      name: t`Mod list`,
      hint: t`Text for Discord or a forum`,
    },
  }
  const reasons = {
    empty: t`No mods to share`,
    'local-only': t`Local archives can't go here`,
  }
  return (
    <TileGrid
      label={t`Where to share`}
      tiles={entries.map((e) => ({
        id: e.id,
        ...tiles[e.id],
        ...(e.reason ? { disabled: reasons[e.reason] } : {}),
      }))}
      lastUsed={lastUsed}
      onChoose={onChoose}
    />
  )
}
