import { useLingui } from '@lingui/react/macro'
import { FileUp, FolderInput, Layers, Link2 } from 'lucide-react'
import { memo } from 'react'
import { GALE, MOD_ORGANIZER_2, R2MODMAN, STARDROP, VORTEX } from '../brand/productNames.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { listNames } from '../i18n/list.ts'
import { type Tile, TileGrid } from './TileGrid.tsx'

const ICON = 32
const MANAGERS = [R2MODMAN, GALE, VORTEX, MOD_ORGANIZER_2]

export type ImportSource = 'link' | 'file' | 'manager' | 'folder'

// The first step of Import: every place a profile can come from, each leading to the same preview.
// modsFolder is whether the game has a Mods folder to import from, and Stardrop, which keeps its profiles beside
// Stardew's Mods folder, comes with it.
// Memoised: the dialog redraws as the mods folder check finishes, and the choices need not.
export const ImportSources = memo(function ImportSourcesView({
  modsFolder,
  onChoose,
}: {
  modsFolder: boolean
  onChoose: (source: ImportSource) => void
}) {
  const { t } = useLingui()
  const managers = modsFolder ? [...MANAGERS, STARDROP] : MANAGERS
  const tiles: Tile<ImportSource>[] = [
    {
      id: 'link',
      icon: <Link2 size={ICON} />,
      name: t`Link or code`,
      hint: t`A Mortar link, a ${{ source: sourceLabel('nexus') }} collection or an ${{ manager: R2MODMAN }} or ${{ otherManager: GALE }} code`,
    },
    {
      id: 'file',
      icon: <FileUp size={ICON} />,
      name: t`File`,
      hint: t`A .mortar file, an .r2z or a ${{ source: sourceLabel('thunderstore') }} modpack`,
    },
    {
      id: 'manager',
      icon: <Layers size={ICON} />,
      name: t`Another mod manager`,
      hint: t`${{ managers: listNames(managers, managers.length) }} on this computer`,
    },
    ...(modsFolder
      ? [
          {
            id: 'folder' as const,
            icon: <FolderInput size={ICON} />,
            name: t`The game's Mods folder`,
            hint: t`Mods already in the game's folder`,
          },
        ]
      : []),
  ]
  return (
    <TileGrid
      label={t`Where the profile comes from`}
      tiles={tiles}
      lastUsed={null}
      onChoose={onChoose}
    />
  )
})
