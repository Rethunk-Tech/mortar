import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { Browser } from '@wailsio/runtime'
import { TriangleAlert } from 'lucide-react'
import { type ReactNode, useEffect } from 'react'
import type {
  Ref,
  Result,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import type { Drift } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  AdoptDriftFolder,
  ForgetDriftEntry,
  KeepDriftChanges,
  RemoveDriftFolder,
  RestoreDriftEntry,
  RevertDriftEntry,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useConsole } from '../console/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { download, type Want } from '../queue/actions.ts'
import { useQueue } from '../queue/store.ts'
import { pendingFor } from '../queue/totals.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useDescribe } from './describe.ts'
import { type Problem, problemsOf, sameId } from './lookup.ts'
import { useMods } from './store.ts'
import { useLocked } from './useLocked.ts'

const ROW_HEIGHT = 38
const ROW_GAP = 6
const VISIBLE_ROWS = 3
const FOCUS_DEBOUNCE_MS = 400

type Row = Problem | { kind: 'drift'; drift: Drift }

const driftRows = (result: Result | null): Row[] =>
  (result?.drift ?? []).map((drift) => ({ kind: 'drift' as const, drift }))

function WhereButtons({ where, addLabel }: { where: Ref; addLabel: string }) {
  const { t } = useLingui()
  const queue = useQueue((s) => s.state.items)
  const profileId = useProfiles((s) => s.openId)
  const { url } = where
  if (!url) {
    return null
  }
  const open = (
    <Button
      size="small"
      color="warning"
      variant="outlined"
      onClick={() => Browser.OpenURL(url).catch(reportUnexpected)}
      sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
    >
      {t`Open page`}
    </Button>
  )
  const github = where.site === 'GitHub' && where.github !== ''
  if (!github && (where.site !== 'Nexus' || where.pageId <= 0)) {
    return open
  }
  const queued = github
    ? pendingFor(queue, profileId, 0, where.github)
    : pendingFor(queue, profileId, where.pageId)
  const want: Want = github
    ? { kind: 'dependency', repo: where.github, name: where.github }
    : {
        kind: 'dependency',
        modId: where.pageId,
        fileId: where.fileId,
        name: where.pageName,
        fileName: where.fileName,
        version: where.version,
      }
  return (
    <>
      {open}
      <Button
        size="small"
        variant="contained"
        color="warning"
        disabled={queued}
        onClick={() => download([want]).catch(reportUnexpected)}
        sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
      >
        {queued ? t`Queued` : (addLabel ?? '')}
      </Button>
    </>
  )
}

function RunErrorButtons({
  runError,
  button,
}: {
  runError: Extract<Problem, { kind: 'runError' }>['runError']
  button: (label: string, onClick: () => void) => ReactNode
}) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const setEnabled = useMods((s) => s.setEnabled)
  const mod = mods.find((m) => m.key === runError.key && sameId(m.uniqueId, runError.uniqueId))
  const openHelp = () => {
    const { game, openId } = useProfiles.getState()
    if (!(game && openId) || runError.runId === '') {
      return
    }
    useConsole.getState().viewRun(game.id, openId, runError.runId)
    useConsole.getState().setHelping(true)
  }
  return (
    <>
      {mod ? button(t`Switch off`, () => setEnabled(mod, false).catch(reportUnexpected)) : null}
      <Button
        size="small"
        color="warning"
        variant="outlined"
        onClick={openHelp}
        sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
      >
        {t`Get help`}
      </Button>
    </>
  )
}

function ListedButtons({
  missing,
  dismiss,
}: {
  missing: Extract<Problem, { kind: 'missing' }>['missing']
  dismiss: (uniqueId: string) => Promise<void>
}) {
  const { t } = useLingui()
  return (
    <>
      {missing.where ? <WhereButtons where={missing.where} addLabel={t`Add`} /> : null}
      <Button
        size="small"
        color="info"
        variant="outlined"
        onClick={() => dismiss(missing.uniqueId).catch(reportUnexpected)}
        sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
      >
        {t`Dismiss`}
      </Button>
    </>
  )
}

function FixButton({ problem }: { problem: Problem }) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const setEnabled = useMods((s) => s.setEnabled)
  const resolve = useMods((s) => s.resolve)
  const dismissAsset = useMods((s) => s.dismissAsset)
  const dismissAbandoned = useMods((s) => s.dismissAbandoned)
  const dismissListed = useMods((s) => s.dismissListed)
  const locked = useLocked()
  const button = (label: string, onClick: () => void) => (
    <Button
      size="small"
      variant="contained"
      color="warning"
      disabled={locked}
      onClick={onClick}
      sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
    >
      {label}
    </Button>
  )
  if (problem.kind === 'runError') {
    return <RunErrorButtons runError={problem.runError} button={button} />
  }
  if (problem.kind === 'duplicate') {
    return button(t`Resolve`, () => resolve(problem.duplicate))
  }
  if (problem.kind === 'broken') {
    const { broken } = problem
    const mod = mods.find((m) => m.key === broken.key && sameId(m.uniqueId, broken.uniqueId))
    const where = broken.replacement
    const replaceName =
      where?.pageName?.trim() ||
      where?.github?.trim() ||
      where?.fileName?.trim() ||
      (where && where.pageId > 0 ? String(where.pageId) : '')
    let replace: ReactNode = null
    if (where && replaceName !== '') {
      replace = <WhereButtons where={where} addLabel={t`Replace with ${replaceName}`} />
    } else if (where?.url) {
      replace = (
        <Button
          size="small"
          color="warning"
          variant="outlined"
          onClick={() => Browser.OpenURL(where.url).catch(reportUnexpected)}
          sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
        >
          {t`Open page`}
        </Button>
      )
    }
    const dismiss =
      broken.status === 'abandoned' ? (
        <Button
          size="small"
          color="info"
          variant="outlined"
          onClick={() => dismissAbandoned(broken.uniqueId).catch(reportUnexpected)}
          sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
        >
          {t`Dismiss`}
        </Button>
      ) : null
    return (
      <>
        {mod ? button(t`Switch off`, () => setEnabled(mod, false).catch(reportUnexpected)) : null}
        {replace}
        {dismiss}
      </>
    )
  }
  if (problem.kind === 'asset') {
    const key = problem.asset.keys?.[0]
    const id = problem.asset.packIds?.[0]
    const mod = mods.find((m) => m.key === key && (id === undefined || sameId(m.uniqueId, id)))
    const dismiss = dismissAsset
    return (
      <>
        {mod ? button(t`Switch off`, () => setEnabled(mod, false).catch(reportUnexpected)) : null}
        {problem.asset.kind === 'edit' ? (
          <Button
            size="small"
            color="info"
            variant="outlined"
            onClick={() => dismiss(problem.asset).catch(reportUnexpected)}
            sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
          >
            {t`Dismiss`}
          </Button>
        ) : null}
      </>
    )
  }
  const { missing } = problem
  if (missing.listed) {
    return <ListedButtons missing={missing} dismiss={dismissListed} />
  }
  if (missing.reason === 'disabled') {
    const off = mods.find((m) => !m.enabled && sameId(m.uniqueId, missing.uniqueId))
    return off ? button(t`Switch on`, () => setEnabled(off, true).catch(reportUnexpected)) : null
  }
  const { where } = missing
  if (!where) {
    return null
  }
  return <WhereButtons where={where} addLabel={t`Add`} />
}

function DriftButtons({ drift }: { drift: Drift }) {
  const { t } = useLingui()
  const load = useMods((s) => s.load)
  const loadProblems = useMods((s) => s.loadProblems)
  const replace = useProfiles((s) => s.replace)
  const locked = useLocked()
  const target = () => {
    const { game, openId } = useProfiles.getState()
    return game && openId ? { game: game.id, id: openId } : null
  }
  const run = (work: () => Promise<unknown>) => {
    work()
      .then(() => Promise.all([load(), loadProblems()]))
      .catch(reportUnexpected)
  }
  const button = (label: string, onClick: () => void) => (
    <Button
      size="small"
      variant="contained"
      color="warning"
      disabled={locked}
      onClick={onClick}
      sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
    >
      {label}
    </Button>
  )
  if (drift.kind === 'unknown') {
    return (
      <>
        {button(t`Adopt`, () =>
          run(async () => {
            const open = target()
            if (!open) {
              return
            }
            replace(await AdoptDriftFolder(open.game, open.id, drift.folder))
          }),
        )}
        {button(t`Remove`, () =>
          run(async () => {
            const open = target()
            if (!open) {
              return
            }
            await RemoveDriftFolder(open.game, open.id, drift.folder)
          }),
        )}
      </>
    )
  }
  if (drift.kind === 'deleted') {
    return (
      <>
        {button(t`Restore`, () =>
          run(async () => {
            const open = target()
            if (!open) {
              return
            }
            replace(await RestoreDriftEntry(open.game, open.id, drift.key))
          }),
        )}
        {button(t`Forget`, () =>
          run(async () => {
            const open = target()
            if (!open) {
              return
            }
            replace(await ForgetDriftEntry(open.game, open.id, drift.key))
          }),
        )}
      </>
    )
  }
  return (
    <>
      {button(t`Keep my changes`, () =>
        run(async () => {
          const open = target()
          if (!open) {
            return
          }
          await KeepDriftChanges(open.game, open.id, drift.key)
        }),
      )}
      {button(t`Revert`, () =>
        run(async () => {
          const open = target()
          if (!open) {
            return
          }
          replace(await RevertDriftEntry(open.game, open.id, drift.key))
        }),
      )}
    </>
  )
}

export function ProblemBar() {
  const { t } = useLingui()
  const describe = useDescribe()
  const result = useMods((s) => s.problems)
  const loadProblems = useMods((s) => s.loadProblems)
  const problems = [...problemsOf(result), ...driftRows(result)]
  const describeDrift = (d: Drift) => {
    if (d.kind === 'unknown') {
      return t`${d.folder} is in this profile's mods folder and is not an installed entry.`
    }
    if (d.kind === 'deleted') {
      return t`${d.key} was removed from this profile's mods folder.`
    }
    return t`${d.key} was changed outside Mortar.`
  }
  useEffect(() => {
    let timer: ReturnType<typeof globalThis.setTimeout> | undefined
    const scan = () => {
      globalThis.clearTimeout(timer)
      timer = globalThis.setTimeout(() => {
        loadProblems().catch(reportUnexpected)
      }, FOCUS_DEBOUNCE_MS)
    }
    const onVisible = () => {
      if (document.visibilityState === 'visible') {
        scan()
      }
    }
    globalThis.addEventListener('focus', scan)
    document.addEventListener('visibilitychange', onVisible)
    return () => {
      globalThis.clearTimeout(timer)
      globalThis.removeEventListener('focus', scan)
      document.removeEventListener('visibilitychange', onVisible)
    }
  }, [loadProblems])
  if (result === null) {
    return (
      <Typography sx={{ mx: 2, mt: 1.25, fontSize: 13, color: 'text.secondary' }}>
        {t`Checking the mods for problems…`}
      </Typography>
    )
  }
  if (problems.length === 0 && !result.unknown) {
    return null
  }
  return (
    <Box sx={{ mx: 2, mt: 1.25, flexShrink: 0 }}>
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          gap: `${ROW_GAP}px`,
          maxHeight: VISIBLE_ROWS * ROW_HEIGHT + (VISIBLE_ROWS - 1) * ROW_GAP,
          overflowY: 'auto',
        }}
      >
        {problems.map((p) => {
          const info =
            (p.kind === 'asset' && p.asset.kind === 'edit') ||
            (p.kind === 'broken' && p.broken.status === 'abandoned') ||
            (p.kind === 'missing' && p.missing.listed) ||
            (p.kind === 'runError' && !p.runError.severe)
          return (
            <Box
              key={JSON.stringify(p)}
              role="alert"
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: 1.25,
                minHeight: ROW_HEIGHT,
                flexShrink: 0,
                pl: 1.5,
                pr: 0.75,
                fontSize: 14,
                bgcolor: info ? 'rgba(56,189,248,0.12)' : 'rgba(243,180,22,0.14)',
                border: info ? '1px solid rgba(56,189,248,0.45)' : '1px solid rgba(243,180,22,0.5)',
                borderRadius: '6px',
              }}
            >
              <Box
                component="span"
                sx={{ display: 'flex', flexShrink: 0, color: info ? 'info.main' : 'warning.main' }}
              >
                <TriangleAlert size={16} aria-hidden={true} />
              </Box>
              <Typography noWrap={true} sx={{ flex: 1, minWidth: 0, fontSize: 14 }}>
                {p.kind === 'drift' ? describeDrift(p.drift) : describe(p)}
              </Typography>
              {p.kind === 'drift' ? <DriftButtons drift={p.drift} /> : <FixButton problem={p} />}
            </Box>
          )
        })}
      </Box>
      {result?.unknown ? (
        <Typography sx={{ mt: 0.5, fontSize: 12, color: 'text.secondary' }}>
          {t`Some checks could not run without a connection, so more problems may show up later.`}
        </Typography>
      ) : null}
    </Box>
  )
}
