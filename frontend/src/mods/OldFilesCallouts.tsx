import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Collapse,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from '@mui/material'
import { useCallback, useEffect, useState } from 'react'
import type {
  OldFiles,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  PendingOldFiles,
  ResolveOldFiles,
  RestoreOldFiles,
  TrashOldFiles,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { ListCallout } from './ListCallout.tsx'
import { LockedReason } from './LockedReason.tsx'
import { useMods } from './store.ts'
import { useLocked } from './useLocked.ts'

const trashedTitle = (n: number) =>
  plural(n, { one: 'Moved # old file to trash', other: 'Moved # old files to trash' })

function OldFilesCallout({
  set,
  game,
  profileId,
  onDone,
  onRestored,
}: {
  set: OldFiles
  game: string
  profileId: string
  onDone: (key: string) => void
  onRestored: () => void
}) {
  const { t } = useLingui()
  const locked = useLocked()
  const [pending, run] = usePending()
  const [shown, setShown] = useState(false)
  const files = set.files ?? []
  const keep = () =>
    run(
      async () => {
        await ResolveOldFiles(game, profileId, set.key, true)
        onDone(set.key)
        await useMods.getState().load()
      },
      { errorTitle: t`Could not keep the old files` },
    )
  const trash = () =>
    run(
      async () => {
        const token = await TrashOldFiles(game, profileId, set.key)
        onDone(set.key)
        useToasts.getState().push({
          kind: 'success',
          title: trashedTitle(files.length),
          action: {
            label: i18n._(msg`Undo`),
            profileId,
            run: () =>
              RestoreOldFiles(game, profileId, token).then(onRestored).catch(reportUnexpected),
          },
        })
      },
      { errorTitle: t`Could not move the old files to trash` },
    )
  const text = plural(files.length, {
    one: `The new version of ${set.label} no longer includes # file. Keep it or delete it?`,
    other: `The new version of ${set.label} no longer includes # files. Keep them or delete them?`,
  })
  return (
    <ListCallout
      text={text}
      actions={
        <>
          <Button variant="text" aria-expanded={shown} onClick={() => setShown(!shown)}>
            {t`Show files`}
          </Button>
          <LockedReason locked={locked}>
            <Button
              variant="text"
              aria-label={t`Keep ${set.label}'s old files`}
              disabled={pending || locked}
              onClick={keep}
            >
              {t`Keep`}
            </Button>
          </LockedReason>
          <LockedReason locked={locked}>
            <Button
              variant="contained"
              color="error"
              aria-label={t`Delete ${set.label}'s old files`}
              disabled={pending || locked}
              onClick={trash}
            >
              {t`Delete`}
            </Button>
          </LockedReason>
        </>
      }
    >
      <Collapse in={shown} unmountOnExit={true}>
        <Box
          tabIndex={0}
          role="region"
          aria-label={t`Files set aside`}
          sx={{ maxHeight: 160, overflowY: 'auto' }}
        >
          {files.map((f) => (
            <Typography
              key={`${f.id}/${f.path}`}
              sx={{ fontSize: 12, fontFamily: 'monospace', overflowWrap: 'anywhere' }}
            >
              {f.path}
            </Typography>
          ))}
        </Box>
      </Collapse>
    </ListCallout>
  )
}

// One callout when a single entry's last update set files aside; with more, one summary callout, so a large
// update cannot push the mod list off the page, and a dialog listing each entry.
export function OldFilesCallouts({ profile }: { profile: Profile }) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const locked = useLocked()
  const [pending, run] = usePending()
  const [reviewing, setReviewing] = useState(false)
  const [found, setFound] = useState<{ id: string; sets: OldFiles[] }>({ id: '', sets: [] })
  const sets = found.id === profile.id ? found.sets : []
  const refetch = useCallback(
    () =>
      PendingOldFiles(game, profile.id)
        .then((list) => setFound({ id: profile.id, sets: list ?? [] }))
        .catch(reportUnexpected),
    [game, profile.id],
  )
  useEffect(() => {
    if (game !== '' && profile.updated !== '') {
      refetch().catch(reportUnexpected)
    }
  }, [game, profile.updated, refetch])
  const done = (key: string) =>
    setFound((f) => ({ ...f, sets: f.sets.filter((s) => s.key !== key) }))
  const callout = (set: OldFiles) => (
    <OldFilesCallout
      key={set.key}
      set={set}
      game={game}
      profileId={profile.id}
      onRestored={() => {
        refetch().catch(reportUnexpected)
      }}
      onDone={done}
    />
  )
  const [only] = sets
  if (only === undefined) {
    if (reviewing) {
      setReviewing(false)
    }
    return null
  }
  if (sets.length === 1 && !reviewing) {
    return callout(only)
  }
  const files = sets.reduce((n, s) => n + (s.files?.length ?? 0), 0)
  const keepAll = () =>
    run(
      async () => {
        for (const set of sets) {
          await ResolveOldFiles(game, profile.id, set.key, true)
          done(set.key)
        }
        setReviewing(false)
        await useMods.getState().load()
      },
      { errorTitle: t`Could not keep the old files` },
    )
  const trashAll = () =>
    run(
      async () => {
        const tokens: string[] = []
        for (const set of sets) {
          tokens.push(await TrashOldFiles(game, profile.id, set.key))
          done(set.key)
        }
        setReviewing(false)
        useToasts.getState().push({
          kind: 'success',
          title: trashedTitle(files),
          action: {
            label: i18n._(msg`Undo`),
            profileId: profile.id,
            run: () =>
              Promise.all(tokens.map((token) => RestoreOldFiles(game, profile.id, token)))
                .then(refetch)
                .catch(reportUnexpected),
          },
        })
      },
      { errorTitle: t`Could not move the old files to trash` },
    )
  const text = t`${plural(sets.length, { one: '# updated mod', other: '# updated mods' })} no longer include ${plural(files, { one: '# file', other: '# files' })} from their old versions. Keep them or delete them?`
  const bulk = (
    <>
      <LockedReason locked={locked}>
        <Button variant="text" disabled={pending || locked} onClick={keepAll}>
          {t`Keep all`}
        </Button>
      </LockedReason>
      <LockedReason locked={locked}>
        <Button variant="contained" color="error" disabled={pending || locked} onClick={trashAll}>
          {t`Delete all`}
        </Button>
      </LockedReason>
    </>
  )
  return (
    <>
      <ListCallout
        text={text}
        actions={
          <>
            <Button variant="text" onClick={() => setReviewing(true)}>
              {t`Review…`}
            </Button>
            {bulk}
          </>
        }
      />
      <Dialog
        open={reviewing}
        onClose={() => setReviewing(false)}
        fullWidth={true}
        maxWidth="md"
        scroll="paper"
      >
        <DialogTitle>{t`Files left by updates`}</DialogTitle>
        <DialogContent dividers={true} sx={{ px: 0, pt: 0 }}>
          {sets.map(callout)}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setReviewing(false)}>{t`Close`}</Button>
          {bulk}
        </DialogActions>
      </Dialog>
    </>
  )
}
