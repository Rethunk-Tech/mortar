import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from '@mui/material'
import { useEffect, useState } from 'react'
import type {
  EverywherePreview,
  EverywhereResult,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  PreviewEverywhere,
  UpdateEverywhere,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { errorMessage } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { usePending } from '../../toasts/usePending.ts'
import { paper } from '../paper.ts'
import { mergePreviews } from './mergePreviews.ts'

function PreviewLists({ preview }: { preview: EverywherePreview }) {
  const { t } = useLingui()
  const affected = preview.affected ?? []
  const skipped = preview.skipped ?? []
  const reasonText = (reason: string) => {
    if (reason === 'pinned') {
      return t`pinned`
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
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
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
}: {
  loadError: string
  preview: EverywherePreview | null
}) {
  const { t } = useLingui()
  if (loadError !== '') {
    return <Typography color="error">{loadError}</Typography>
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
  const [loadError, setLoadError] = useState('')
  const spec = mods.map((m) => `${m.id}\0${m.newKey}`).join('\n')
  useEffect(() => {
    if (!open || game === '' || spec === '') {
      return
    }
    setPreview(null)
    setLoadError('')
    const items = spec.split('\n').map((row) => {
      const [id] = row.split('\0')
      return { id: id ?? '' }
    })
    Promise.all(items.map((m) => PreviewEverywhere(game, m.id)))
      .then((parts) => setPreview(mergePreviews(parts)))
      .catch((err: unknown) => setLoadError(errorMessage(err)))
  }, [open, game, spec])
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
      })
      onClose()
    })
  }
  return (
    <Dialog
      open={open}
      onClose={pending ? undefined : onClose}
      transitionDuration={0}
      slotProps={{ paper: { sx: { ...paper.sx, width: 420, maxWidth: 'calc(100% - 32px)' } } }}
    >
      <DialogTitle>{t`Update in all profiles (${n})`}</DialogTitle>
      <DialogContent>
        <PreviewBody loadError={loadError} preview={preview} />
      </DialogContent>
      <DialogActions sx={{ bgcolor: 'background.paper' }}>
        <Button onClick={onClose} disabled={pending}>
          {t`Cancel`}
        </Button>
        <Button variant="contained" onClick={apply} disabled={pending || n === 0}>
          {t`Update`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
