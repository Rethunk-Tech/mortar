import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { useLoaded } from '@rethunk/react-load-hooks'
import type {
  EverywherePreview,
  EverywhereResult,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  PreviewEverywhere,
  UpdateEverywhere,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { lowerFirst } from '../../i18n/lowerFirst.ts'
import { openProfileOf, useProfiles } from '../../profiles/store.ts'
import { download } from '../../queue/actions.ts'
import { ConfirmationDialog } from '../../shell/dialog/ConfirmationDialog.tsx'
import { ErrorRetry } from '../../shell/ErrorRetry.tsx'
import { space } from '../../theme/density.ts'
import { type InlineError, inlineError } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { usePending } from '../../toasts/usePending.ts'
import { sameId, visibleUpdates } from '../lookup.ts'
import { useUpdates } from '../updates.ts'
import { mergePreviews } from './mergePreviews.ts'
import { withOptionalLoaded } from './wants.ts'

function PreviewLists({ preview }: { preview: EverywherePreview }) {
  const { t } = useLingui()
  const affected = preview.affected ?? []
  const skipped = preview.skipped ?? []
  const reasonText = (reason: string) => {
    if (reason === 'pinned') {
      return lowerFirst(t`Pinned`)
    }
    if (reason === 'skip-version') {
      return t`this version skipped`
    }
    if (reason === 'locked') {
      return t`game running`
    }
    return reason
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}>
      <Typography sx={{ fontSize: 'base.fontSize' }}>{t`Will update`}</Typography>
      {affected.length === 0 ? (
        <Typography color="text.secondary">{t`No eligible profiles.`}</Typography>
      ) : (
        affected.map((row) => (
          <Typography key={row.profileId} sx={{ fontSize: 'base.fontSize' }}>
            {row.name}
          </Typography>
        ))
      )}
      {skipped.length > 0 ? (
        <>
          <Typography sx={{ fontSize: 'base.fontSize' }}>{t`Skipped`}</Typography>
          {skipped.map((row) => (
            <Typography
              key={row.profileId}
              sx={{ fontSize: 'base.fontSize' }}
              color="text.secondary"
            >
              {t`${{ name: row.name }}: ${{ reason: reasonText(row.reason) }}`}
            </Typography>
          ))}
        </>
      ) : null}
    </Box>
  )
}

function PreviewBody({
  loadError,
  preview,
  onRetry,
}: {
  loadError: InlineError | null
  preview: EverywherePreview | null
  onRetry: () => void
}) {
  const { t } = useLingui()
  if (loadError) {
    return <ErrorRetry error={loadError} onRetry={onRetry} />
  }
  if (preview === null) {
    return <Typography>{t`Checking profiles…`}</Typography>
  }
  return <PreviewLists preview={preview} />
}

function mergeResults(parts: EverywhereResult[]): EverywhereResult {
  return {
    updated: parts.flatMap((p) => p.updated ?? []),
    skipped: parts.flatMap((p) => p.skipped ?? []),
  }
}

export function EverywhereDialog({
  open,
  game,
  mods,
  onClose,
}: {
  open: boolean
  game: string
  mods: { id: string; newKey: string }[]
  onClose: () => void
}) {
  const { t } = useLingui()
  const [pending, run] = usePending()
  const spec = mods.map((m) => `${m.id}\0${m.newKey}`).join('\n')
  const {
    data: preview,
    error,
    reload: load,
  } = useLoaded<EverywherePreview | null>(
    open && game !== '' && spec !== ''
      ? () =>
          Promise.all(
            spec.split('\n').map((row) => PreviewEverywhere(game, row.split('\0')[0] ?? '')),
          ).then(mergePreviews)
      : null,
    [open, game, spec],
    null,
  )
  const loadError = error === null ? null : inlineError(error)
  const n = preview?.affected?.length ?? 0
  // The new version is not in the store yet: queue its download once; when it lands, the queue updates the other profiles.
  const downloadFirst = async () => {
    const profile = openProfileOf(useProfiles.getState())
    const updates = visibleUpdates(useUpdates.getState().updates, profile)
    const wants = await Promise.all(
      mods.map(async (m) => {
        const update = updates.find((u) => sameId(u.id, m.id))
        if (!update) {
          return []
        }
        const [main, ...rest] = await withOptionalLoaded(game, update, profile)
        return main ? [{ ...main, everywhere: true }, ...rest] : []
      }),
    )
    await download(wants.flat())
  }
  const apply = () => {
    if (preview && !preview.inStore) {
      run(async () => {
        await downloadFirst()
        onClose()
      })
      return
    }
    run(async () => {
      const parts = await Promise.all(
        mods.map((m) => UpdateEverywhere(game, m.id, m.newKey || 'latest')),
      )
      const merged = mergeResults(parts)
      useToasts.getState().push({
        kind: 'success',
        title: plural(
          { count: merged.updated?.length ?? 0 },
          { one: 'Updated in # profile', other: 'Updated in # profiles' },
        ),
        changes: (merged.updated ?? []).map((hit) => hit.change ?? ''),
      })
      onClose()
    })
  }
  return (
    <ConfirmationDialog
      open={open}
      title={t`Update in all profiles?`}
      confirmLabel={t`Update`}
      busy={pending}
      confirmDisabled={n === 0}
      onCancel={onClose}
      onConfirm={apply}
    >
      <PreviewBody loadError={loadError} preview={preview} onRetry={load} />
    </ConfirmationDialog>
  )
}
