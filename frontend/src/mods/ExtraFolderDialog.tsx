import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Button, Typography } from '@mui/material'
import { FolderOpen } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { ExtraFolderMods } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useInstall } from '../install/store.ts'
import { useNav } from '../nav/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { partitionPreview, selectedFolders, toggled } from './libraryRows.ts'
import { PreviewPick } from './PreviewPick.tsx'
import { usePreviewRows } from './usePreviewRows.ts'

export function ExtraFolderDialog({
  open,
  game,
  folder,
  onClose,
}: {
  open: boolean
  game: string
  folder: string
  onClose: () => void
}) {
  const { t } = useLingui()
  const fetchRows = useCallback(() => ExtraFolderMods(game), [game])
  const { mods, loading, error, reload } = usePreviewRows(open, fetchRows)
  const [off, setOff] = useState<ReadonlySet<string>>(new Set())
  useEffect(() => {
    if (open) {
      setOff(new Set())
    }
  }, [open])
  const { pickable, blocked } = partitionPreview(mods)
  const chosen = selectedFolders(pickable, off)
  const empty = !loading && error === '' && mods.length === 0
  return (
    <ConfirmDialog
      open={open}
      title={t`Add from the extra mods folder`}
      confirmLabel={t`Add ${plural(chosen.length, { one: '# mod', other: '# mods' })}`}
      confirmDisabled={loading || chosen.length === 0}
      maxWidth={520}
      onCancel={onClose}
      onConfirm={() => {
        useInstall.getState().installFromExtraFolder(chosen).catch(reportUnexpected)
        onClose()
      }}
    >
      {loading ? <LoadingRow>{t`Reading ${folder}…`}</LoadingRow> : null}
      {error === '' ? null : (
        <>
          <Typography sx={{ color: 'error.main' }}>{error}</Typography>
          <Button onClick={reload} sx={{ mt: 1 }}>
            {t`Retry`}
          </Button>
        </>
      )}
      {empty ? (
        <EmptyState
          compact={true}
          icon={<FolderOpen />}
          title={t`No mods in ${folder}`}
          action={
            <Button
              onClick={() => {
                onClose()
                useNav.getState().openGameSettings()
              }}
            >
              {t`Change folder`}
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
