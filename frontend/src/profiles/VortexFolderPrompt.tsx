import { useLingui } from '@lingui/react/macro'
import { useCallback, useEffect, useState } from 'react'
import type { VortexInventory } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/migrate/models.ts'
import { ExternalVortexContents } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { VORTEX } from '../brand/productNames.ts'
import { useGameName } from '../games/info.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { ManagerFolderPrompt } from './ManagerFolderPrompt.tsx'

function useVortexNote(game: string, contents: VortexInventory | null): string {
  const { t } = useLingui()
  const gameName = useGameName(game)
  const folder = contents?.folder ?? ''
  const found = (contents?.games ?? []).map((g) => `${g.id} (${g.profiles})`).join(', ')
  if (!folder) {
    return t`Use ${{ manager: VORTEX }} from another folder?`
  }
  if (!found) {
    return t`The ${{ manager: VORTEX }} folder ${{ path: folder }} has no profiles.`
  }
  return t`This ${{ manager: VORTEX }} folder has no profiles for ${{ game: gameName }}: found ${found}.`
}

// Shown when no Vortex profile matches the game: says what the Vortex folder does hold, and lets the user
// point Mortar at another one.
export function VortexFolderPrompt({ game, onChosen }: { game: string; onChosen: () => void }) {
  const { t } = useLingui()
  const [contents, setContents] = useState<VortexInventory | null>(null)
  const load = useCallback(() => {
    ExternalVortexContents().then(setContents).catch(reportUnexpected)
  }, [])
  useEffect(load, [load])
  const note = useVortexNote(game, contents)
  return (
    <ManagerFolderPrompt
      settingKey="vortexFolder"
      label={t`${{ manager: VORTEX }} data folder`}
      note={note}
      onChosen={() => {
        load()
        onChosen()
      }}
    />
  )
}
