import { useNav } from '../nav/store.ts'
import { GameModsDialog } from './GameModsDialog.tsx'
import { useGameModsDialog } from './gameModsDialog.ts'
import { useProfiles } from './store.ts'

export function GameModsDialogHost() {
  const game = useGameModsDialog((s) => s.game)
  const close = useGameModsDialog((s) => s.close)
  const openProfile = useProfiles((s) => s.open)
  return (
    <GameModsDialog
      open={game !== ''}
      game={game}
      onClose={close}
      onImported={(id) => {
        openProfile(id)
        useNav.getState().closeProfiles()
      }}
    />
  )
}
