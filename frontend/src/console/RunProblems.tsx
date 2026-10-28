import type { I18n } from '@lingui/core'
import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { useCallback } from 'react'
import type { SMAPIProblem } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import { RunProblems as FetchRunProblems } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import {
  DismissedTokens,
  DismissRunProblem,
  RestoreDismissed,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/service.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useBrowseView } from '../browse/view.ts'
import { useTab } from '../game/tab.ts'
import { useGameLoader } from '../games/info.ts'
import { lowerFirst } from '../i18n/lowerFirst.ts'
import { useLoader } from '../loader/store.ts'
import { localId } from '../mods/dependents.ts'
import { sameId } from '../mods/lookup.ts'
import { useMods } from '../mods/store.ts'
import { useUpdates } from '../mods/updates.ts'
import { useLocked } from '../mods/useLocked.ts'
import { download } from '../queue/actions.ts'
import { refWant } from '../queue/refWant.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { useLoaded } from '../shell/useLoaded.ts'
import { calloutFill, calloutLine } from '../theme/callout.ts'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { matchMod, stillApplies } from './runProblemsLive.ts'
import { runProblemMod, runProblemToken } from './runProblemToken.ts'

function findMod(problem: SMAPIProblem): Mod | undefined {
  return matchMod(problem, useMods.getState().mods)
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
      title: i18n._(msg`Mortar does not know where to get ${id}`),
      action: {
        label: i18n._(msg`Search in Browse`),
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

async function applyFix(i18n: I18n, problem: SMAPIProblem, game: string) {
  switch (problem.fix) {
    case 'updateLoader':
      await useLoader.getState().install(game)
      return
    case 'remove': {
      const mod = findMod(problem)
      if (mod) {
        await useMods.getState().remove(mod)
      }
      return
    }
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
      const off = await useMods.getState().setEnabled(mod, false)
      if (off.length === 0) {
        return
      }
      useToasts.getState().push({
        kind: 'success',
        title: i18n._(msg`Disabled ${{ names: mod.name }}`),
        action: {
          label: i18n._(msg`Undo`),
          run: () => useMods.getState().setEnabledMany(off, true),
        },
      })
      return
    }
    case 'removeDuplicate': {
      const dup = matchMod(problem, useMods.getState().problems?.duplicates ?? [])
      if (dup) {
        useMods.getState().resolve(dup)
      }
      return
    }
    default:
      return
  }
}

function ProblemRow({
  problem,
  contained,
  game,
  onDismiss,
}: {
  problem: SMAPIProblem
  contained: boolean
  game: string
  onDismiss: (problem: SMAPIProblem) => void
}) {
  const { t, i18n } = useLingui()
  const loaderName = useGameLoader(game) || lowerFirst(t`Mod loader`)
  const locked = useLocked()
  const lockHint = t`Stop the game to change mods.`
  let label = ''
  if (problem.fix === 'installDependency') {
    label = t`Add`
  } else if (problem.fix === 'update') {
    label = t`Update`
  } else if (problem.fix === 'disable') {
    label = t`Disable`
  } else if (problem.fix === 'removeDuplicate') {
    label = t`Resolve`
  } else if (problem.fix === 'updateLoader') {
    label = t`Update ${{ loader: loaderName }}`
  } else if (problem.fix === 'remove') {
    label = t`Remove`
  }
  const run = () => {
    applyFix(i18n, problem, game).catch(reportUnexpected)
  }
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'flex-start',
        gap: space.gap,
        fontSize: 'medium.fontSize',
      }}
    >
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
      <Button
        size="small"
        color="inherit"
        onClick={() => onDismiss(problem)}
        sx={{ flexShrink: 0 }}
      >
        {t`Dismiss`}
      </Button>
    </Box>
  )
}

const NO_PROBLEMS: SMAPIProblem[] = []
const NO_TOKENS: string[] = []

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
  const { data: found } = useLoaded(
    () =>
      FetchRunProblems(game, profile, run).then(
        (rows) => rows ?? [],
        () => NO_PROBLEMS,
      ),
    [game, profile, run],
    NO_PROBLEMS,
  )
  const mods = useMods((s) => s.mods)
  const locked = useLocked()
  // The problems service keeps dismissals with the profile's other dismissed problems; this strip reads them itself,
  // as the Problems page does, and again after each change it makes.
  const { data: dismissed, reload: reloadDismissed } = useLoaded(
    () =>
      DismissedTokens(game, profile).then(
        (tokens) => tokens ?? NO_TOKENS,
        () => NO_TOKENS,
      ),
    [game, profile],
    NO_TOKENS,
  )
  const dismiss = useCallback(
    (p: SMAPIProblem) => {
      DismissRunProblem(game, profile, p.kind, runProblemMod(p))
        .then(reloadDismissed)
        .catch(reportUnexpected)
    },
    [game, profile, reloadDismissed],
  )
  const applying = found.filter((p) => stillApplies(p, mods))
  const live = applying.filter((p) => !dismissed.includes(runProblemToken(p)))
  const hidden = applying.length - live.length
  const bulk = live.filter(
    (p) => p.fix === 'installDependency' || p.fix === 'update' || p.fix === 'disable',
  )
  if (live.length === 0) {
    return hidden > 0 ? (
      <Box
        sx={{
          mx: space.gutter,
          mb: space.gap,
          display: 'flex',
          alignItems: 'center',
          gap: space.gap,
        }}
      >
        <Typography sx={{ fontSize: 'base.fontSize', color: 'text.secondary' }}>
          {plural(hidden, {
            one: '# problem in this run is dismissed',
            other: '# problems in this run are dismissed',
          })}
        </Typography>
        <Button
          size="small"
          color="inherit"
          onClick={() =>
            Promise.all(applying.map((p) => RestoreDismissed(game, profile, runProblemToken(p))))
              .then(reloadDismissed)
              .catch(reportUnexpected)
          }
        >
          {t`Restore`}
        </Button>
      </Box>
    ) : null
  }
  return (
    <Box
      sx={{
        mx: space.gutter,
        mb: space.gap,
        px: space.pad,
        py: space.gap,
        display: 'flex',
        flexDirection: 'column',
        gap: space.gap,
        bgcolor: calloutFill('warning'),
        border: '1px solid',
        borderColor: calloutLine('warning'),
        borderRadius: '6px',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
        <Typography
          sx={{ fontSize: 'base.fontSize', fontWeight: 700, flex: 1 }}
        >{t`Problems in this run`}</Typography>
        {bulk.length > 0 ? (
          <DisabledReason title={t`Stop the game to change mods.`} disabled={locked}>
            <Button
              size="small"
              variant="contained"
              color="warning"
              disabled={locked}
              onClick={() =>
                Promise.all(bulk.map((problem) => applyFix(i18n, problem, game))).catch(
                  reportUnexpected,
                )
              }
            >
              {t`Fix all`}
            </Button>
          </DisabledReason>
        ) : null}
      </Box>
      {live.map((problem) => (
        <ProblemRow
          key={`${problem.kind}-${problem.modId}-${problem.detail}`}
          problem={problem}
          contained={false}
          game={game}
          onDismiss={dismiss}
        />
      ))}
    </Box>
  )
}
