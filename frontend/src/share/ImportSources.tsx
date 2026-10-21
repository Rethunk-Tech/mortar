import { useLingui } from '@lingui/react/macro'
import { FileUp, FolderInput, Layers, Link2 } from 'lucide-react'
import { type Tile, TileGrid } from './TileGrid.tsx'

const ICON = 32

export type ImportSource = 'link' | 'file' | 'manager' | 'folder'

// The first step of Import: every place a profile can come from, each leading to the same preview.
export function ImportSources({ onChoose }: { onChoose: (source: ImportSource) => void }) {
  const { t } = useLingui()
  const tiles: Tile<ImportSource>[] = [
    {
      id: 'link',
      icon: <Link2 size={ICON} />,
      name: t`A link or code`,
      hint: t`A Mortar link, a Nexus collection or an r2modman or Gale code`,
    },
    {
      id: 'file',
      icon: <FileUp size={ICON} />,
      name: t`A file`,
      hint: t`A .mortar file, an .r2z or a Thunderstore modpack`,
    },
    {
      id: 'manager',
      icon: <Layers size={ICON} />,
      name: t`Another mod manager`,
      hint: t`r2modman, Gale, Vortex, Mod Organizer 2 or Stardrop on this computer`,
    },
    {
      id: 'folder',
      icon: <FolderInput size={ICON} />,
      name: t`The game's Mods folder`,
      hint: t`Mods already in the game's folder`,
    },
  ]
  return (
    <TileGrid
      label={t`Where the profile comes from`}
      tiles={tiles}
      lastUsed={null}
      onChoose={onChoose}
    />
  )
}
