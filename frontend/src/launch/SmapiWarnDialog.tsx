import { useLingui } from '@lingui/react/macro'
import { useGameName } from '../games/info.ts'
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
  return (
    <ConfirmDialog
      open={open}
      maxWidth={440}
      title={t`Steam will still start SMAPI`}
      body={t`${gameName}'s Steam launch options run SMAPI in place of the game. Steam will still start SMAPI unless that launch option is removed. Mortar will not change it.`}
      confirmLabel={t`Play without mods`}
      onCancel={onClose}
      onConfirm={onPlay}
    />
  )
}
