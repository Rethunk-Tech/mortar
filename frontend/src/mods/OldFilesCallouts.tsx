import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Collapse, Typography } from '@mui/material'
import { Files } from 'lucide-react'
import { memo, useState } from 'react'
import type { OldFiles } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  PendingOldFiles,
  ResolveOldFiles,
  RestoreOldFiles,
  TrashOldFiles,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { AppDialog, DialogFooter, ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { useLoaded } from '../shell/useLoaded.ts'
import { MONO } from '../theme/theme.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { ListCallout } from './ListCallout.tsx'
import { LockedReason } from './LockedReason.tsx'
import { useMods } from './store.ts'
import { useLocked } from './useLocked.ts'

const trashedTitle = (n: number) =>
  plural({ count: n }, { one: 'Moved # old file to trash', other: 'Moved # old files to trash' })

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
  const lost = plural({ count: files.length }, { one: '# file', other: '# files' })
  const question = plural(
    { count: files.length },
    { one: 'Keep it or delete it?', other: 'Keep them or delete them?' },
  )
  const text = `${t`The new version of ${{ name: set.label }} no longer includes ${lost}.`} ${question}`
  return (
    <ListCallout
      text={text}
      actions={
        <>
          <Button
            size="small"
            variant="text"
            aria-expanded={shown}
            onClick={() => setShown(!shown)}
          >
            {t`Show files`}
          </Button>
          <LockedReason locked={locked}>
            <Button
              size="small"
              variant="text"
              aria-label={t`Keep ${{ name: set.label }}'s old files`}
              disabled={pending || locked}
              onClick={keep}
            >
              {t`Keep`}
            </Button>
          </LockedReason>
          <LockedReason locked={locked}>
            <Button
              size="small"
              variant="contained"
              color="error"
              aria-label={t`Delete ${{ name: set.label }}'s old files`}
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
              sx={{ fontSize: 'small.fontSize', fontFamily: MONO, overflowWrap: 'anywhere' }}
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
export const OldFilesCallouts = memo(function OldFilesCalloutsView({
  profileId,
  stamp,
}: {
  profileId: string
  stamp: string
}) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const locked = useLocked()
  const [pending, run] = usePending()
  const [reviewing, setReviewing] = useState(false)
  const {
    data: sets,
    setData: setSets,
    reload: refetch,
  } = useLoaded<OldFiles[]>(
    game !== '' && stamp !== ''
      ? () => PendingOldFiles(game, profileId).then((list) => list ?? [])
      : null,
    [game, profileId, stamp],
    [],
    reportUnexpected,
  )
  const done = (key: string) => setSets((all) => all.filter((s) => s.key !== key))
  const callout = (set: OldFiles) => (
    <OldFilesCallout
      key={set.key}
      set={set}
      game={game}
      profileId={profileId}
      onRestored={refetch}
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
          await ResolveOldFiles(game, profileId, set.key, true)
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
          tokens.push(await TrashOldFiles(game, profileId, set.key))
          done(set.key)
        }
        setReviewing(false)
        useToasts.getState().push({
          kind: 'success',
          title: trashedTitle(files),
          action: {
            label: i18n._(msg`Undo`),
            profileId,
            run: () =>
              Promise.all(tokens.map((token) => RestoreOldFiles(game, profileId, token)))
                .then(refetch)
                .catch(reportUnexpected),
          },
        })
      },
      { errorTitle: t`Could not move the old files to trash` },
    )
  const mods = plural({ count: sets.length }, { one: '# updated mod', other: '# updated mods' })
  const oldFiles = plural({ count: files }, { one: '# file', other: '# files' })
  const text = t`${mods} no longer include ${oldFiles} from their old versions. Keep them or delete them?`
  const bulk = (
    <>
      <LockedReason locked={locked}>
        <Button size="small" variant="text" disabled={pending || locked} onClick={keepAll}>
          {t`Keep all`}
        </Button>
      </LockedReason>
      <LockedReason locked={locked}>
        <Button
          size="small"
          variant="contained"
          color="error"
          disabled={pending || locked}
          onClick={trashAll}
        >
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
            <Button size="small" variant="text" onClick={() => setReviewing(true)}>
              {t`Review…`}
            </Button>
            {bulk}
          </>
        }
      />
      <AppDialog
        open={reviewing}
        onClose={() => setReviewing(false)}
        size="lg"
        title={t`Files left by updates`}
        icon={
          <ToneIcon tone="neutral">
            <Files size={22} />
          </ToneIcon>
        }
        footer={
          <DialogFooter>
            <Button onClick={() => setReviewing(false)}>{t`Close`}</Button>
            {bulk}
          </DialogFooter>
        }
      >
        {sets.map(callout)}
      </AppDialog>
    </>
  )
})
