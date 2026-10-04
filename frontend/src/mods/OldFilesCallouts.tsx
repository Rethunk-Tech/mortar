import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Collapse, Typography } from '@mui/material'
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
          title: plural(files.length, {
            one: 'Moved # old file to trash',
            other: 'Moved # old files to trash',
          }),
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
              key={`${f.uniqueId}/${f.path}`}
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

// One callout per entry whose last update set files aside; it refreshes whenever the profile changes.
export function OldFilesCallouts({ profile }: { profile: Profile }) {
  const game = useProfiles((s) => s.game?.id ?? '')
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
  return sets.map((set) => (
    <OldFilesCallout
      key={set.key}
      set={set}
      game={game}
      profileId={profile.id}
      onRestored={() => {
        refetch().catch(reportUnexpected)
      }}
      onDone={(key) => setFound((f) => ({ ...f, sets: f.sets.filter((s) => s.key !== key) }))}
    />
  ))
}
