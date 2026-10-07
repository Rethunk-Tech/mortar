import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { useCallback, useEffect, useState } from 'react'
import type {
  EverywherePreview,
  EverywhereResult,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  PreviewEverywhere,
  UpdateEverywhere,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { ErrorRetry } from '../../shell/ErrorRetry.tsx'
import { space } from '../../theme/density.ts'
import { type InlineError, inlineError } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { usePending } from '../../toasts/usePending.ts'
import { mergePreviews } from './mergePreviews.ts'

function PreviewLists({ preview }: { preview: EverywherePreview }) {
  const { t } = useLingui()
  const affected = preview.affected ?? []
  const skipped = preview.skipped ?? []
  const reasonText = (reason: string) => {
    if (reason === 'pinned') {
      return t`Pinned`.toLowerCase()
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
      <Typography sx={{ fontSize: 13 }}>{t`Will update`}</Typography>
      {affected.length === 0 ? (
        <Typography color="text.secondary">{t`No eligible profiles.`}</Typography>
      ) : (
        affected.map((row) => (
          <Typography key={row.profileId} sx={{ fontSize: 13 }}>
            {row.name}
          </Typography>
        ))
      )}
      {skipped.length > 0 ? (
        <>
          <Typography sx={{ fontSize: 13 }}>{t`Skipped`}</Typography>
          {skipped.map((row) => (
            <Typography key={row.profileId} sx={{ fontSize: 13 }} color="text.secondary">
              {t`${row.name}: ${reasonText(row.reason)}`}
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
  const [preview, setPreview] = useState<EverywherePreview | null>(null)
  const [pending, run] = usePending()
  const [loadError, setLoadError] = useState<InlineError | null>(null)
  const spec = mods.map((m) => `${m.id}\0${m.newKey}`).join('\n')
  const load = useCallback(() => {
    if (game === '' || spec === '') {
      return
    }
    setPreview(null)
    setLoadError(null)
    const items = spec.split('\n').map((row) => {
      const [id] = row.split('\0')
      return { id: id ?? '' }
    })
    Promise.all(items.map((m) => PreviewEverywhere(game, m.id)))
      .then((parts) => setPreview(mergePreviews(parts)))
      .catch((err: unknown) => setLoadError(inlineError(err)))
  }, [game, spec])
  useEffect(() => {
    if (open) {
      load()
    }
  }, [open, load])
  const n = preview?.affected?.length ?? 0
  const apply = () => {
    run(async () => {
      const parts = await Promise.all(
        mods.map((m) => UpdateEverywhere(game, m.id, m.newKey || 'latest')),
      )
      const merged = mergeResults(parts)
      useToasts.getState().push({
        kind: 'success',
        title: t`${plural(merged.updated?.length ?? 0, { one: 'Updated in # profile', other: 'Updated in # profiles' })}`,
        changes: (merged.updated ?? []).map((hit) => hit.change ?? ''),
      })
      onClose()
    })
  }
  return (
    <ConfirmDialog
      open={open}
      title={t`Update in all profiles (${n})`}
      confirmLabel={t`Update`}
      busy={pending}
      confirmDisabled={n === 0}
      maxWidth={420}
      onCancel={onClose}
      onConfirm={apply}
    >
      <PreviewBody loadError={loadError} preview={preview} onRetry={load} />
    </ConfirmDialog>
  )
}
