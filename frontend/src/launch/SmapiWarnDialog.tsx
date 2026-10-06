import { useLingui } from '@lingui/react/macro'
import { useGameInfo, useGameName } from '../games/info.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'

export function SmapiWarnDialog({
  open,
  onClose,
  onPlay,
}: {
  open: boolean
  onClose: () => void
  onPlay: () => void
}) {
  const { t } = useLingui()
  const gameName = useGameName()
  const loader = useGameInfo()?.loader ?? ''
  return (
    <ConfirmDialog
      open={open}
      maxWidth={440}
      title={t`Steam will still start ${loader}`}
      body={t`${gameName}'s Steam launch options run ${loader} in place of the game. Steam will still start ${loader} unless that launch option is removed. Mortar will not change it.`}
      confirmLabel={t`Play without mods`}
      onCancel={onClose}
      onConfirm={onPlay}
    />
  )
}
