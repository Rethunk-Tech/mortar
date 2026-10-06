import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  List,
  ListItem,
  ListItemText,
  ListSubheader,
  Skeleton,
} from '@mui/material'
import { ShieldCheck } from 'lucide-react'
import { type ReactNode, useCallback, useEffect, useState } from 'react'
import type { HealthFinding } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  ProfileHealth,
  RepairProfile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { absoluteWhen } from '../i18n/when.ts'
import { openSettings } from '../nav/store.ts'
import { download } from '../queue/actions.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { reportError, reportUnexpected } from '../toasts/report.ts'
import { downloadWantsForEntries, type UndoEntry } from '../toasts/undo.ts'
import { historyLabel } from './historyLabel.ts'
import { useProfiles } from './store.ts'

const KINDS = ['missing', 'drift', 'snapshot', 'journal', 'unused'] as const

function useKindTitles(): Record<string, string> {
  const { t } = useLingui()
  return {
    missing: t`Missing from the store`,
    drift: t`Changed outside Mortar`,
    snapshot: t`History that cannot be read`,
    journal: t`Unfinished launch`,
    unused: t`Not used by any profile`,
  }
}

// findingText is the one-line sentence for a finding, built here so every kind reads through Lingui.
function findingText(f: HealthFinding): string {
  const name = f.items?.[0] ?? ''
  switch (f.kind) {
    case 'missing':
      return f.repair === ''
        ? i18n._(msg`${name} is missing from the store, and Mortar cannot download it again`)
        : i18n._(msg`${name} is missing from the store`)
    case 'drift':
      return f.cause === 'deleted'
        ? i18n._(msg`The folder of ${name} was deleted outside Mortar`)
        : i18n._(msg`Files of ${name} changed outside Mortar`)
    case 'snapshot': {
      const change = f.event ? historyLabel(f.event) : ''
      const when = absoluteWhen(f.at ?? '', i18n.locale)
      return f.cause === 'configs'
        ? i18n._(msg`The saved change ${change} from ${when} is missing its config files`)
        : i18n._(msg`Could not read the saved change ${change} from ${when}`)
    }
    case 'journal':
      return i18n._(msg`A launch ended without Mortar putting the game folder back as it was`)
    default: {
      const count = f.items?.length ?? 0
      return plural(count, {
        one: '# stored mod is not used by any profile',
        other: '# stored mods are not used by any profile',
      })
    }
  }
}

function FindingGroups({
  findings,
  busy,
  queued,
  onRepair,
}: {
  findings: HealthFinding[]
  busy: boolean
  queued: Set<string>
  onRepair: (f: HealthFinding) => void
}) {
  const { t } = useLingui()
  const titles = useKindTitles()
  return KINDS.filter((kind) => findings.some((f) => f.kind === kind)).map((kind) => (
    <List
      key={kind}
      disablePadding={true}
      subheader={<ListSubheader disableGutters={true}>{titles[kind]}</ListSubheader>}
    >
      {findings
        .filter((f) => f.kind === kind)
        .map((f) => (
          <ListItem
            key={f.id}
            disableGutters={true}
            secondaryAction={
              f.repair === '' ? null : (
                <Button
                  size="small"
                  disabled={busy || queued.has(f.id)}
                  onClick={() => onRepair(f)}
                >
                  {queued.has(f.id) ? t`Queued` : t`Repair`}
                </Button>
              )
            }
          >
            <ListItemText
              primary={findingText(f)}
              secondary={kind === 'unused' || kind === 'journal' ? f.items?.join(', ') : null}
              slotProps={{ secondary: { noWrap: true, title: f.items?.join(', ') } }}
            />
          </ListItem>
        ))}
    </List>
  ))
}

// Reverting replaces edited files, so it is asked about even inside Repair all; the unreadable history goes with it.
function confirmTitle(chosen: HealthFinding[]): string {
  const reverts = chosen.filter((f) => f.repair === 'revert')
  if (reverts.length === 0) {
    return i18n._(msg`Delete the unreadable changes from history?`)
  }
  const name = reverts[0]?.items?.[0] ?? ''
  return reverts.length === 1
    ? i18n._(msg`Revert ${name} to the installed copy?`)
    : plural(reverts.length, { one: 'Revert # mod?', other: 'Revert # mods?' })
}

function confirmBody(chosen: HealthFinding[]): string {
  const revert = i18n._(msg`Your edits to its files are replaced with the installed copy.`)
  const drop = i18n._(msg`The profile itself stays as it is.`)
  const reverts = chosen.some((f) => f.repair === 'revert')
  const drops = chosen.some((f) => f.repair === 'drop-snapshot')
  if (reverts && drops) {
    return `${revert} ${i18n._(msg`The unreadable changes are also deleted from history.`)}`
  }
  return reverts ? revert : drop
}

export function HealthDialog({
  profileId,
  open,
  onClose,
}: {
  profileId: string
  open: boolean
  onClose: () => void
}) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const [findings, setFindings] = useState<HealthFinding[] | null>(null)
  const [failed, setFailed] = useState(false)
  const [busy, setBusy] = useState(false)
  const [queued, setQueued] = useState<Set<string>>(new Set())
  const [confirm, setConfirm] = useState<HealthFinding[] | null>(null)
  const load = useCallback(() => {
    setFailed(false)
    setFindings(null)
    ProfileHealth(game, profileId)
      .then((next) => setFindings(next ?? []))
      .catch(() => setFailed(true))
  }, [game, profileId])
  useEffect(() => {
    if (open && game !== '') {
      setQueued(new Set())
      load()
    }
  }, [open, game, load])

  const repair = async (chosen: HealthFinding[]) => {
    setBusy(true)
    try {
      const fetch = chosen.filter((f) => f.repair === 'download')
      if (fetch.length > 0) {
        const entries = fetch.flatMap((f) => (f.entries ?? []) as UndoEntry[])
        if (await download(downloadWantsForEntries(entries))) {
          setQueued((cur) => new Set([...cur, ...fetch.map((f) => f.id)]))
        }
      }
      const here = chosen.filter((f) =>
        ['restore', 'revert', 'drop-snapshot', 'recover'].includes(f.repair),
      )
      if (here.length > 0) {
        useProfiles.getState().replace(
          await RepairProfile(
            game,
            profileId,
            here.map((f) => f.id),
          ),
        )
        load()
      }
      if (chosen.some((f) => f.repair === 'cleanup')) {
        onClose()
        openSettings('storage')
      }
    } catch (e) {
      reportError(t`Could not repair this profile`)(e)
      load()
    } finally {
      setBusy(false)
    }
  }
  const ask = (chosen: HealthFinding[]) => {
    if (chosen.some((f) => f.repair === 'drop-snapshot' || f.repair === 'revert')) {
      setConfirm(chosen)
    } else {
      repair(chosen).catch(reportUnexpected)
    }
  }
  const repairable = (findings ?? []).filter((f) => f.repair !== '' && !queued.has(f.id))

  let body: ReactNode
  if (failed) {
    body = (
      <Box role="alert" sx={{ display: 'flex', alignItems: 'center', gap: 1, py: 1.5 }}>
        {t`Could not check this profile`}
        <Button variant="outlined" onClick={load}>
          {t`Retry`}
        </Button>
      </Box>
    )
  } else if (findings === null) {
    body = <Skeleton height={44} />
  } else if (findings.length === 0) {
    body = (
      <EmptyState compact={true} icon={<ShieldCheck size={28} />} title={t`Everything looks fine`}>
        {t`Mortar found nothing to repair in this profile.`}
      </EmptyState>
    )
  } else {
    body = (
      <FindingGroups findings={findings} busy={busy} queued={queued} onRepair={(f) => ask([f])} />
    )
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      slotProps={{ paper: { sx: { minWidth: 440, maxWidth: 'calc(100vw - 64px)' } } }}
    >
      <DialogTitle>{t`Check this profile`}</DialogTitle>
      <DialogContent>{body}</DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t`Close`}</Button>
        {findings !== null && findings.length > 0 ? (
          <Button
            variant="contained"
            disabled={busy || repairable.length === 0}
            onClick={() => ask(repairable)}
          >
            {t`Repair all`}
          </Button>
        ) : null}
      </DialogActions>
      <ConfirmDialog
        open={confirm !== null}
        title={confirmTitle(confirm ?? [])}
        body={confirmBody(confirm ?? [])}
        confirmLabel={confirm?.some((f) => f.repair === 'revert') ? t`Revert` : t`Delete`}
        color="error"
        busy={busy}
        onCancel={() => setConfirm(null)}
        onConfirm={() => {
          const chosen = confirm ?? []
          setConfirm(null)
          repair(chosen).catch(reportUnexpected)
        }}
      />
    </Dialog>
  )
}
