import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { FolderOpen } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { ExtraFolderMods } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useInstall } from '../install/store.ts'
import { useNav } from '../nav/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { ErrorRetry } from '../shell/ErrorRetry.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { useFolderEvent } from '../shell/useFolderEvent.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { partitionPreview, selectedFolders, toggled } from './libraryRows.ts'
import { PreviewPick } from './PreviewPick.tsx'
import { usePreviewRows } from './usePreviewRows.ts'

export function ExtraFolderDialog({
  open,
  game,
  onClose,
}: {
  open: boolean
  game: string
  onClose: () => void
}) {
  const { t } = useLingui()
  const fetchRows = useCallback(() => ExtraFolderMods(game), [game])
  const { mods, loading, error, reload } = usePreviewRows(open, fetchRows)
  useFolderEvent('library:extra-folder', game, () => open && reload())
  const [off, setOff] = useState<ReadonlySet<string>>(new Set())
  useEffect(() => {
    if (open) {
      setOff(new Set())
    }
  }, [open])
  const { pickable, blocked } = partitionPreview(mods)
  const chosen = selectedFolders(pickable, off)
  const empty = !loading && error === null && mods.length === 0
  return (
    <ConfirmDialog
      open={open}
      title={t`Add from the extra mods folder`}
      confirmLabel={plural(chosen.length, { one: 'Add # mod', other: 'Add # mods' })}
      confirmDisabled={loading || chosen.length === 0}
      maxWidth={520}
      onCancel={onClose}
      onConfirm={() => {
        useInstall.getState().installFromExtraFolder(chosen).catch(reportUnexpected)
        onClose()
      }}
    >
      {loading ? <LoadingRow>{t`Reading the extra mods folders…`}</LoadingRow> : null}
      {error === null ? null : <ErrorRetry error={error} onRetry={reload} />}
      {empty ? (
        <EmptyState
          compact={true}
          icon={<FolderOpen />}
          title={t`No mods in the extra mods folders`}
          action={
            <Button
              onClick={() => {
                onClose()
                useNav.getState().openGameSettings()
              }}
            >
              {t`Change folders`}
            </Button>
          }
        >
          {t`Mods you put there show up here to add.`}
        </EmptyState>
      ) : null}
      {mods.length === 0 ? null : (
        <PreviewPick
          pickable={pickable}
          blocked={blocked}
          off={off}
          onToggle={(f) => setOff(toggled(off, f))}
        />
      )}
    </ConfirmDialog>
  )
}
