import { useLingui } from '@lingui/react/macro'
import { Code, Frame, Image, Package, Radio, Shirt, Tv, Volume2 } from 'lucide-react'
import type { ReactNode } from 'react'
import type { AssetKind } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/components/models.ts'
import { type Tile, TileGrid } from '../share/TileGrid.tsx'

const ICON_PX = 32

const KIND_ICONS: Record<string, ReactNode> = {
  suits: <Shirt size={ICON_PX} />,
  sounds: <Volume2 size={ICON_PX} />,
  posters: <Image size={ICON_PX} />,
  paintings: <Frame size={ICON_PX} />,
  television: <Tv size={ICON_PX} />,
  boombox: <Radio size={ICON_PX} />,
}

// A Thunderstore Team-Name as people say it: x753-More_Suits is More Suits.
const packageTitle = (pkg: string) => pkg.slice(pkg.indexOf('-') + 1).replaceAll('_', ' ')

/** The wizard's first step when the game offers asset kinds: a code mod from the template, or one made only of files.
 * Picking a card moves on. */
export function KindStep({
  kinds,
  kind,
  choose,
}: {
  kinds: AssetKind[]
  kind: string
  choose: (kind: string) => void
}) {
  const { t } = useLingui()
  const tiles: Tile<string>[] = [
    {
      id: '',
      icon: <Code size={ICON_PX} />,
      name: t`Code`,
      hint: t`A plugin in C#, built with the .NET SDK from the game's template.`,
    },
    ...kinds.map((k) => ({
      id: k.id,
      icon: KIND_ICONS[k.id] ?? <Package size={ICON_PX} />,
      name: k.label,
      hint: t`Files that ${{ name: packageTitle(k.dependency) }} loads. No SDK needed.`,
    })),
  ]
  return (
    <TileGrid
      label={t`Kind of mod`}
      tiles={tiles}
      lastUsed={null}
      selected={kind}
      onChoose={choose}
      padded={false}
    />
  )
}
