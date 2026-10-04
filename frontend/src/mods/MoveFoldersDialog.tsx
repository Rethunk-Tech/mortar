import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { useEffect, useState } from 'react'
import type {
  GameModPreview,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  DismissGameModsFolder,
  MoveGameModsFolders,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { formatOutcomeDetail } from '../profiles/gameModsFormat.ts'
import { useProfiles } from '../profiles/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { partitionPreview, selectedFolders, toggled } from './libraryRows.ts'
import { PreviewPick } from './PreviewPick.tsx'
import { useMods } from './store.ts'
import { useLocked } from './useLocked.ts'

export function MoveFoldersDialog({
  open,
  game,
  profile,
  mods,
  onDismiss,
  onClose,
}: {
  open: boolean
  game: string
  profile: Profile
  mods: GameModPreview[]
  onDismiss: (folder: string) => void
  onClose: (moved: boolean) => void
}) {
  const { t } = useLingui()
  const locked = useLocked()
  const [off, setOff] = useState<ReadonlySet<string>>(new Set())
  const [busy, run] = usePending()
  useEffect(() => {
    if (open) {
      setOff(new Set())
    }
  }, [open])
  const { pickable, blocked } = partitionPreview(mods)
  const chosen = selectedFolders(pickable, off)
  const move = () =>
    run(
      async () => {
        const res = await MoveGameModsFolders(game, profile.id, chosen)
        await Promise.all([useProfiles.getState().load(game), useMods.getState().load()])
        const detail = formatOutcomeDetail(res.outcomes ?? [])
        const body = [
          res.skipped > 0 ? t`${res.skipped} skipped` : '',
          res.failed > 0 ? t`${res.failed} failed` : '',
        ]
          .filter((part) => part !== '')
          .join(' · ')
        useToasts.getState().push({
          kind: res.failed > 0 ? 'warning' : 'success',
          title: plural(res.imported, { one: 'Moved # mod', other: 'Moved # mods' }),
          ...(body === '' ? {} : { body }),
          ...(detail === '' ? {} : { detail }),
        })
        onClose(true)
      },
      { errorTitle: t`Could not move the mods` },
    )
  return (
    <ConfirmDialog
      open={open}
      title={t`Move mods into ${profile.name}`}
      body={t`The folders are moved out of the game's Mods folder and into this profile.`}
      confirmLabel={t`Move ${plural(chosen.length, { one: '# mod', other: '# mods' })}`}
      confirmDisabled={chosen.length === 0 || locked}
      busy={busy}
      maxWidth={560}
      onCancel={() => onClose(false)}
      onConfirm={move}
    >
      <PreviewPick
        pickable={pickable}
        blocked={blocked}
        off={off}
        onToggle={(f) => setOff(toggled(off, f))}
        action={(row) => (
          <Button
            size="small"
            variant="text"
            color="inherit"
            aria-label={t`Don't ask again about ${row.folder ?? ''}`}
            disabled={busy}
            onClick={() => {
              DismissGameModsFolder(game, row.folder ?? '')
                .then(() => onDismiss(row.folder ?? ''))
                .catch(reportError(t`Could not dismiss that folder`))
            }}
          >
            {t`Don't ask again`}
          </Button>
        )}
      />
    </ConfirmDialog>
  )
}
