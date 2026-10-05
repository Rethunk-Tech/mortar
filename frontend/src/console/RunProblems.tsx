import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { useEffect, useState } from 'react'
import type { SMAPIProblem } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import { RunProblems as FetchRunProblems } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useBrowseView } from '../browse/view.ts'
import { useTab } from '../game/tab.ts'
import { localId } from '../mods/dependents.ts'
import { sameId } from '../mods/lookup.ts'
import { useMods } from '../mods/store.ts'
import { useUpdates } from '../mods/updates.ts'
import { useLocked } from '../mods/useLocked.ts'
import { download } from '../queue/actions.ts'
import { refWant } from '../queue/refWant.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { calloutFill, calloutLine } from '../theme/callout.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { duplicateCopies } from './runProblemsFix.ts'
import { stillApplies } from './runProblemsLive.ts'

function findMod(problem: SMAPIProblem): Mod | undefined {
  return useMods
    .getState()
    .mods.find(
      (m) =>
        (problem.modId !== '' && sameId(localId(m.id), problem.modId)) ||
        (problem.modName !== '' && m.name === problem.modName),
    )
}

async function installDependency(i18n: I18n, id: string) {
  if (id === '') {
    return
  }
  await useMods.getState().loadProblems()
  const missing = (useMods.getState().problems?.missing ?? []).find((m) =>
    sameId(localId(m.id), id),
  )
  const want = missing?.where ? refWant(missing.where, 'dependency') : null
  if (!want) {
    useToasts.getState().push({
      kind: 'error',
      title: i18n._(msg`Mortar doesn't know where to get ${id}`),
      action: {
        label: i18n._(msg`Search Nexus`),
        run: () => {
          useBrowseView.getState().setPendingQuery(id)
          useTab.getState().setTab('browse')
        },
      },
    })
    return
  }
  await download([want])
}

async function applyFix(i18n: I18n, problem: SMAPIProblem) {
  switch (problem.fix) {
    case 'installDependency':
      await installDependency(i18n, problem.dependency ?? '')
      return
    case 'update':
      useTab.getState().setTab('mods')
      useMods.getState().showUpdates()
      useUpdates.getState().setReviewing(true)
      return
    case 'disable': {
      const mod = findMod(problem)
      if (!mod) {
        return
      }
      await useMods.getState().setEnabled(mod, false)
      useToasts.getState().push({
        kind: 'success',
        title: i18n._(msg`Disabled ${{ names: mod.name }}`),
        action: {
          label: i18n._(msg`Undo`),
          run: () => useMods.getState().setEnabled(mod, true),
        },
      })
      return
    }
    case 'removeDuplicate': {
      const mods = useMods
        .getState()
        .mods.filter(
          (m) =>
            (problem.modId !== '' && sameId(localId(m.id), problem.modId)) ||
            (problem.modName !== '' && m.name === problem.modName),
        )
      const { remove } = duplicateCopies(mods)
      if (remove.length === 1 && remove[0]) {
        await useMods.getState().remove(remove[0])
      } else if (remove.length > 1) {
        await useMods.getState().removeMany(remove)
      }
      return
    }
    default:
      return
  }
}

function ProblemRow({ problem, contained }: { problem: SMAPIProblem; contained: boolean }) {
  const { t, i18n } = useLingui()
  const locked = useLocked()
  const lockHint = t`Stop the game to change mods.`
  const [confirmDup, setConfirmDup] = useState(false)
  let label = ''
  if (problem.fix === 'installDependency') {
    label = t`Install`
  } else if (problem.fix === 'update') {
    label = t`Update`
  } else if (problem.fix === 'disable') {
    label = t`Disable`
  } else if (problem.fix === 'removeDuplicate') {
    label = t`Remove duplicate`
  }
  const copies = duplicateCopies(
    useMods((s) => s.mods).filter(
      (m) =>
        (problem.modId !== '' && sameId(localId(m.id), problem.modId)) ||
        (problem.modName !== '' && m.name === problem.modName),
    ),
  )
  const run = () => {
    if (problem.fix === 'removeDuplicate') {
      setConfirmDup(true)
      return
    }
    applyFix(i18n, problem).catch(reportUnexpected)
  }
  return (
    <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1, fontSize: 14 }}>
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Box sx={{ fontWeight: 600 }}>{problem.modName || problem.modId}</Box>
        <Box sx={{ color: 'text.secondary', mt: 0.25, whiteSpace: 'normal' }}>{problem.detail}</Box>
      </Box>
      {label ? (
        <DisabledReason title={lockHint} disabled={locked}>
          <Button
            size="small"
            variant={contained ? 'contained' : 'outlined'}
            color="warning"
            disabled={locked}
            onClick={run}
            sx={{ flexShrink: 0 }}
          >
            {label}
          </Button>
        </DisabledReason>
      ) : null}
      <ConfirmDialog
        open={confirmDup}
        color="error"
        title={t`Remove duplicate?`}
        body={
          copies.keep
            ? t`Keep ${copies.keep.name} and remove ${copies.remove.map((m) => m.name).join(', ')}.`
            : t`Remove ${{ name: copies.remove.map((m) => m.name).join(', ') }}`
        }
        confirmLabel={t`Remove`}
        onCancel={() => setConfirmDup(false)}
        onConfirm={() => {
          setConfirmDup(false)
          applyFix(i18n, problem).catch(reportUnexpected)
        }}
      />
    </Box>
  )
}

export function RunProblemsStrip({
  game,
  profile,
  run,
}: {
  game: string
  profile: string
  run: string
}) {
  const { t, i18n } = useLingui()
  const [found, setFound] = useState<SMAPIProblem[]>([])
  useEffect(() => {
    FetchRunProblems(game, profile, run).then(
      (rows) => setFound(rows ?? []),
      () => setFound([]),
    )
  }, [game, profile, run])
  const mods = useMods((s) => s.mods)
  const live = found.filter((p) => stillApplies(p, mods))
  const bulk = live.filter(
    (p) => p.fix === 'installDependency' || p.fix === 'update' || p.fix === 'disable',
  )
  if (live.length === 0) {
    return null
  }
  return (
    <Box
      sx={{
        mx: 2,
        mb: 1,
        px: 1.5,
        py: 1,
        display: 'flex',
        flexDirection: 'column',
        gap: 1,
        bgcolor: calloutFill('warning'),
        border: '1px solid',
        borderColor: calloutLine('warning'),
        borderRadius: '6px',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <Typography
          sx={{ fontSize: 13, fontWeight: 700, flex: 1 }}
        >{t`Problems in this run`}</Typography>
        {bulk.length > 0 ? (
          <Button
            size="small"
            variant="contained"
            color="warning"
            onClick={() =>
              Promise.all(bulk.map((problem) => applyFix(i18n, problem))).catch(reportUnexpected)
            }
          >
            {t`Fix all`}
          </Button>
        ) : null}
      </Box>
      {live.map((problem) => (
        <ProblemRow
          key={`${problem.kind}-${problem.modId}-${problem.detail}`}
          problem={problem}
          contained={false}
        />
      ))}
    </Box>
  )
}
