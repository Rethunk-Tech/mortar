import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Tooltip, Typography } from '@mui/material'
import { useEffect, useState } from 'react'
import type { SMAPIProblem } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'
import { RunProblems as FetchRunProblems } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { useTab } from '../game/tab.ts'
import { sameId } from '../mods/lookup.ts'
import { useMods } from '../mods/store.ts'
import { useUpdates } from '../mods/updates.ts'
import { useLocked } from '../mods/useLocked.ts'
import { download, type Want } from '../queue/actions.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { stillApplies } from './runProblemsLive.ts'

function findMod(problem: SMAPIProblem): Mod | undefined {
  return useMods
    .getState()
    .mods.find(
      (m) =>
        (problem.modId !== '' && sameId(m.uniqueId, problem.modId)) ||
        (problem.modName !== '' && m.name === problem.modName),
    )
}

async function installDependency(uniqueId: string) {
  if (uniqueId === '') {
    return
  }
  await useMods.getState().loadProblems()
  const missing = (useMods.getState().problems?.missing ?? []).find((m) =>
    sameId(m.uniqueId, uniqueId),
  )
  const where = missing?.where
  if (!where) {
    return
  }
  const want: Want =
    where.site === 'GitHub'
      ? { kind: 'dependency', repo: where.github, name: where.github }
      : {
          kind: 'dependency',
          modId: where.pageId,
          fileId: where.fileId,
          latest: true,
          name: where.pageName,
          fileName: where.fileName,
          version: where.version,
        }
  await download([want])
}

async function applyFix(i18n: I18n, problem: SMAPIProblem) {
  switch (problem.fix) {
    case 'installDependency':
      await installDependency(problem.dependency ?? '')
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
        title: i18n._(msg`Switched off ${mod.name}`),
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
            (problem.modId !== '' && sameId(m.uniqueId, problem.modId)) ||
            (problem.modName !== '' && m.name === problem.modName),
        )
      const extras = mods.length > 1 ? mods.slice(1) : mods
      const [first, ...rest] = extras
      if (rest.length === 0 && first) {
        await useMods.getState().remove(first)
      } else if (extras.length > 1) {
        await useMods.getState().removeMany(extras)
      }
      return
    }
    default:
      return
  }
}

function ProblemRow({ problem }: { problem: SMAPIProblem }) {
  const { t, i18n } = useLingui()
  const locked = useLocked()
  const lockHint = t`Stop the game to change mods.`
  let label = ''
  if (problem.fix === 'installDependency') {
    label = t`Install`
  } else if (problem.fix === 'update') {
    label = t`Update`
  } else if (problem.fix === 'disable') {
    label = t`Switch off`
  } else if (problem.fix === 'removeDuplicate') {
    label = t`Remove duplicate`
  }
  return (
    <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1, fontSize: 14 }}>
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Box sx={{ fontWeight: 600 }}>{problem.modName || problem.modId}</Box>
        <Box sx={{ color: 'text.secondary', mt: 0.25, whiteSpace: 'normal' }}>{problem.detail}</Box>
      </Box>
      {label ? (
        <Tooltip title={locked ? lockHint : ''}>
          <span>
            <Button
              size="small"
              variant="contained"
              color="warning"
              disabled={locked}
              onClick={() => applyFix(i18n, problem).catch(reportUnexpected)}
              sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
            >
              {label}
            </Button>
          </span>
        </Tooltip>
      ) : null}
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
  const { t } = useLingui()
  const [found, setFound] = useState<SMAPIProblem[]>([])
  useEffect(() => {
    FetchRunProblems(game, profile, run).then(
      (rows) => setFound(rows ?? []),
      () => setFound([]),
    )
  }, [game, profile, run])
  const mods = useMods((s) => s.mods)
  const live = found.filter((p) => stillApplies(p, mods))
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
        bgcolor: 'rgba(243,180,22,0.14)',
        border: '1px solid rgba(243,180,22,0.5)',
        borderRadius: '6px',
      }}
    >
      <Typography sx={{ fontSize: 13, fontWeight: 700 }}>{t`Problems in this run`}</Typography>
      {live.map((problem) => (
        <ProblemRow key={`${problem.kind}-${problem.modId}-${problem.detail}`} problem={problem} />
      ))}
    </Box>
  )
}
