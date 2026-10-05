import { useLingui } from '@lingui/react/macro'
import { useGameName } from '../games/info.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { useLaunch } from './store.ts'

export function StopDialog({
  open,
  game,
  onClose,
}: {
  open: boolean
  game: string
  onClose: () => void
}) {
  const { t } = useLingui()
  const gameName = useGameName(game)
  const stop = useLaunch((s) => s.stop)
  return (
    <ConfirmDialog
      open={open}
      maxWidth={420}
      color="error"
      title={t`Stop the game?`}
      body={t`${gameName} will close now, and any progress since your last save is lost.`}
      confirmLabel={t`Stop game`}
      onCancel={onClose}
      onConfirm={() => {
        onClose()
        stop(game)
      }}
    />
  )
}
