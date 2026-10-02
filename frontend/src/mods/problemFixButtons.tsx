import { useLingui } from '@lingui/react/macro'
import { Button, Menu, MenuItem } from '@mui/material'
import { Browser } from '@wailsio/runtime'
import { ChevronDown } from 'lucide-react'
import { type ReactNode, useState } from 'react'
import type {
  Ref,
  SettingHint,
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
import { type Problem, sameId } from './lookup.ts'
import { useMods } from './store.ts'
import { useLocked } from './useLocked.ts'

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
        latest: true,
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

function BrokenFixButtons({
  broken,
  button,
}: {
  broken: Extract<Problem, { kind: 'broken' }>['broken']
  button: (label: string, onClick: () => void) => ReactNode
}) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const setEnabled = useMods((s) => s.setEnabled)
  const dismissAbandoned = useMods((s) => s.dismissAbandoned)
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

function SettingButtons({ setting }: { setting: SettingHint }) {
  const { t } = useLingui()
  const setConfigValue = useMods((s) => s.setConfigValue)
  const dismissSetting = useMods((s) => s.dismissSetting)
  const locked = useLocked()
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)
  const values = setting.suggested ?? []
  const apply = (value: string) => {
    setAnchorEl(null)
    setConfigValue(setting, value).catch(reportUnexpected)
  }
  if (values.length === 0) {
    return null
  }
  const first = values[0] ?? ''
  // A blank picker value means "detect it for me", which reads better than an empty name.
  const label = (value: string) => (value === '' ? t`Set to automatic` : t`Set to ${value}`)
  return (
    <>
      <Button
        size="small"
        variant="contained"
        color="warning"
        disabled={locked}
        onClick={() => apply(first)}
        sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
      >
        {label(first)}
      </Button>
      {values.length > 1 ? (
        <>
          <Button
            size="small"
            variant="contained"
            color="warning"
            aria-label={t`More setting values`}
            disabled={locked}
            onClick={(event) => setAnchorEl(event.currentTarget)}
            sx={{ minWidth: 28, width: 28, height: 28, px: 0, flexShrink: 0 }}
          >
            <ChevronDown size={15} aria-hidden={true} />
          </Button>
          <Menu anchorEl={anchorEl} open={Boolean(anchorEl)} onClose={() => setAnchorEl(null)}>
            {values.map((value) => (
              <MenuItem key={value} onClick={() => apply(value)}>
                {label(value)}
              </MenuItem>
            ))}
          </Menu>
        </>
      ) : null}
      <Button
        size="small"
        color="info"
        variant="outlined"
        onClick={() => dismissSetting(setting).catch(reportUnexpected)}
        sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
      >
        {t`Dismiss`}
      </Button>
    </>
  )
}

export function FixButton({ problem }: { problem: Problem }) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const setEnabled = useMods((s) => s.setEnabled)
  const resolve = useMods((s) => s.resolve)
  const dismissAsset = useMods((s) => s.dismissAsset)
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
    return <BrokenFixButtons broken={problem.broken} button={button} />
  }
  if (problem.kind === 'setting') {
    return <SettingButtons setting={problem.setting} />
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

export function DriftButtons({ drift }: { drift: Drift }) {
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
