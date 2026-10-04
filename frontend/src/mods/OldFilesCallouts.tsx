import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Collapse, Typography } from '@mui/material'
import { useEffect, useState } from 'react'
import type {
  OldFiles,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  PendingOldFiles,
  ResolveOldFiles,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { ListCallout } from './ListCallout.tsx'
import { useMods } from './store.ts'
import { useLocked } from './useLocked.ts'

function OldFilesCallout({
  set,
  game,
  profileId,
  onDone,
}: {
  set: OldFiles
  game: string
  profileId: string
  onDone: (key: string) => void
}) {
  const { t } = useLingui()
  const locked = useLocked()
  const [pending, run] = usePending()
  const [shown, setShown] = useState(false)
  const files = set.files ?? []
  const resolve = (keep: boolean) =>
    run(async () => {
      await ResolveOldFiles(game, profileId, set.key, keep)
      onDone(set.key)
      await useMods.getState().load()
    })
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
          <DisabledReason title={t`Stop the game to change mods.`} disabled={locked}>
            <Button variant="outlined" disabled={pending || locked} onClick={() => resolve(true)}>
              {t`Keep`}
            </Button>
          </DisabledReason>
          <DisabledReason title={t`Stop the game to change mods.`} disabled={locked}>
            <Button variant="contained" disabled={pending || locked} onClick={() => resolve(false)}>
              {t`Delete`}
            </Button>
          </DisabledReason>
        </>
      }
    >
      <Collapse in={shown} unmountOnExit={true}>
        <Box sx={{ maxHeight: 160, overflowY: 'auto' }}>
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
  useEffect(() => {
    if (game === '' || profile.updated === '') {
      return
    }
    let live = true
    PendingOldFiles(game, profile.id)
      .then((list) => live && setFound({ id: profile.id, sets: list ?? [] }))
      .catch(reportUnexpected)
    return () => {
      live = false
    }
  }, [game, profile.id, profile.updated])
  return sets.map((set) => (
    <OldFilesCallout
      key={set.key}
      set={set}
      game={game}
      profileId={profile.id}
      onDone={(key) => setFound((f) => ({ ...f, sets: f.sets.filter((s) => s.key !== key) }))}
    />
  ))
}
