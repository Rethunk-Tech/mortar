import { useLingui } from '@lingui/react/macro'
import { Download, FolderInput, type LucideIcon } from 'lucide-react'
import { type ReactNode, useMemo, useState } from 'react'
import { useNav } from '../nav/store.ts'
import { openImport } from '../share/store.ts'
import { GameModsDialog } from './GameModsDialog.tsx'
import { ImportWizard } from './ImportWizard.tsx'
import { PackImportDialog } from './PackImportDialog.tsx'
import { hasThunderstore } from './packImport.ts'
import { useProfiles } from './store.ts'

export interface ImportEntry {
  key: string
  label: string
  icon: LucideIcon
  run: () => void
}

// The ways to bring a profile in, with the dialogs they open; `dialogs` must be rendered beside the menu that runs them.
export function useProfileImport(game: string): { entries: ImportEntry[]; dialogs: ReactNode } {
  const { t } = useLingui()
  const openProfile = useProfiles((s) => s.open)
  const thunderstore = hasThunderstore(useProfiles((s) => s.game))
  const [gameMods, setGameMods] = useState(false)
  const [wizard, setWizard] = useState(false)
  const [packImport, setPackImport] = useState(false)
  const [packPath, setPackPath] = useState('')
  const entries: ImportEntry[] = [
    {
      key: 'game-mods',
      label: t`From the game's Mods folder…`,
      icon: FolderInput,
      run: () => setGameMods(true),
    },
    { key: 'link', label: t`From a link or file…`, icon: Download, run: () => openImport() },
    {
      key: 'other-manager',
      label: t`From another mod manager…`,
      icon: Download,
      run: () => setWizard(true),
    },
  ]
  // One element until a dialog's state changes, so the closed ones are skipped when the menu around them renders.
  const dialogs = useMemo(
    () => (
      <>
        <ImportWizard
          open={wizard}
          game={game}
          onClose={() => setWizard(false)}
          onPickPack={(path) => {
            setPackPath(path)
            setPackImport(true)
          }}
          onOwnCode={
            thunderstore
              ? () => {
                  setPackPath('')
                  setPackImport(true)
                }
              : null
          }
        />
        <PackImportDialog
          open={packImport}
          game={game}
          initialPath={packPath}
          onClose={() => setPackImport(false)}
        />
        <GameModsDialog
          open={gameMods}
          game={game}
          onClose={() => setGameMods(false)}
          onImported={(id) => {
            openProfile(id)
            useNav.getState().closeProfiles()
          }}
        />
      </>
    ),
    [game, gameMods, wizard, packImport, packPath, thunderstore, openProfile],
  )
  return { entries, dialogs }
}
