import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  LinearProgress,
} from '@mui/material'
import { useTheme } from '@mui/material/styles'
import { CircleCheck, CircleX, Copy, FileArchive, TriangleAlert } from 'lucide-react'
import { useState } from 'react'
import type {
  Check,
  Report,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/doctor/models.ts'
import {
  Doctor,
  RepairNativeHosts,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/support/service.ts'
import { routeGame, useNav } from '../../nav/store.ts'
import { useProfiles } from '../../profiles/store.ts'
import { saveDiagnostics } from '../../shell/saveDiagnostics.ts'
import { reportError } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { usePending } from '../../toasts/usePending.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'

const detail = { color: 'var(--mortar-ink-soft)' }
const repairFix = 'Repair'

function diagnosticKind(id: string): string {
  const cut = id.indexOf(':')
  return cut === -1 ? id : id.slice(0, cut)
}

function StatusIcon({ status }: { status: string }) {
  const { error, warning, success } = useTheme().palette
  if (status === 'fail') {
    return <CircleX size={16} color={error.main} aria-hidden={true} />
  }
  if (status === 'warn') {
    return <TriangleAlert size={16} color={warning.main} aria-hidden={true} />
  }
  return <CircleCheck size={16} color={success.main} aria-hidden={true} />
}

interface Group {
  kind: string
  status: string
  details: string[]
  repair: Check | undefined
}

const RANK: Record<string, number> = { pass: 0, warn: 1, fail: 2 }

// One row per kind of check (all browsers under "Browser extension"), showing the worst status.
function groupChecks(checks: Check[]): Group[] {
  const groups = new Map<string, Group>()
  for (const c of checks) {
    const kind = diagnosticKind(c.id)
    const g = groups.get(kind) ?? { kind, status: 'pass', details: [], repair: undefined }
    if ((RANK[c.status] ?? 0) > (RANK[g.status] ?? 0)) {
      g.status = c.status
    }
    g.details.push(c.detail)
    if (c.fix === repairFix && !g.repair) {
      g.repair = c
    }
    groups.set(kind, g)
  }
  return [...groups.values()]
}

function useCheckTitle() {
  const { t } = useLingui()
  return (kind: string) => {
    switch (kind) {
      case 'mortar':
        return t`Mortar version`
      case 'dataDir':
        return t`Data folder`
      case 'game':
        return t`Games`
      case 'nxm':
        return t`Nexus download links`
      case 'nativeHost':
        return t`Browser extension`
      case 'settings':
        return t`Settings file`
      case 'settingsCopies':
        return t`Settings backups`
      case 'profile':
      case 'profiles':
        return t`Profiles`
      case 'control':
        return t`Command-line control`
      case 'disk':
        return t`Disk space`
      case 'store':
        return t`Mod store`
      default:
        return kind
    }
  }
}

function Diagnostics() {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const title = useCheckTitle()
  const game = useNav((s) => routeGame(s.route) ?? '')
  const profile = useProfiles((s) => s.openId)
  const [open, setOpen] = useState(false)
  const [report, setReport] = useState<Report | null>(null)
  const [busy, runChecks] = usePending()
  const [repairing, runRepair] = usePending()
  const [failed, setFailed] = useState(false)
  const run = () =>
    runChecks(() =>
      Doctor()
        .then((r) => {
          setFailed(false)
          setReport(r ?? { checks: [] })
        })
        .catch(() => {
          setFailed(true)
          setReport(null)
        }),
    )
  const repair = () => runRepair(() => RepairNativeHosts().then(run))
  const groups = groupChecks(report?.checks ?? [])
  const copy = () => {
    const text = groups
      .map((g) => `${title(g.kind)} (${g.status})\n${g.details.join('\n')}`)
      .join('\n\n')
    navigator.clipboard.writeText(`${text}\n`).then(
      () => push({ kind: 'success', title: t`Report copied` }),
      (err: unknown) => reportError(t`Could not copy the report`)(err),
    )
  }
  return (
    <SettingsSection title={t`Diagnostics`}>
      <SettingRow
        label={t`Check Mortar`}
        description={t`Checks the data folder, games, Nexus download links and the browser extension.`}
      >
        <Button
          variant="outlined"
          onClick={() => {
            setOpen(true)
            run()
          }}
        >
          {t`Run checks…`}
        </Button>
      </SettingRow>
      <SettingRow
        label={t`Save diagnostics`}
        description={t`Writes a zip with logs and settings, personal paths removed, to attach to a bug report.`}
      >
        <Button
          variant="outlined"
          startIcon={<FileArchive size={16} />}
          onClick={() => saveDiagnostics(game, profile)}
        >
          {t`Save…`}
        </Button>
      </SettingRow>
      <Dialog
        open={open}
        onClose={() => setOpen(false)}
        maxWidth="sm"
        fullWidth={true}
        scroll="paper"
      >
        <DialogTitle>{t`Diagnostics`}</DialogTitle>
        <DialogContent dividers={true} sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
          {busy && !report ? <LinearProgress /> : null}
          {failed ? (
            <Box role="alert" sx={{ display: 'flex', alignItems: 'center', gap: 1, fontSize: 15 }}>
              {t`Could not run diagnostics`}
              <Button size="small" variant="outlined" disabled={busy} onClick={run}>
                {t`Run again`}
              </Button>
            </Box>
          ) : null}
          {!busy && report && groups.length === 0 ? (
            <Box sx={{ fontSize: 15 }}>{t`All checks passed`}</Box>
          ) : null}
          {groups.map((g) => (
            <Box key={g.kind} sx={{ display: 'flex', gap: 1.25, alignItems: 'flex-start' }}>
              <Box sx={{ pt: '3px' }}>
                <StatusIcon status={g.status} />
              </Box>
              <Box sx={{ flex: 1, minWidth: 0 }}>
                <Box sx={{ fontSize: 15, fontWeight: 600 }}>{title(g.kind)}</Box>
                {g.details.map((d) => (
                  <Box key={d} sx={{ ...detail, fontSize: 13, overflowWrap: 'anywhere' }}>
                    {d}
                  </Box>
                ))}
              </Box>
              {g.repair ? (
                <Button
                  size="small"
                  variant="outlined"
                  disabled={repairing || busy}
                  onClick={repair}
                >
                  {t`Repair`}
                </Button>
              ) : null}
            </Box>
          ))}
        </DialogContent>
        <DialogActions>
          <Button startIcon={<Copy size={16} />} disabled={!report} onClick={copy}>
            {t`Copy report`}
          </Button>
          <Button
            startIcon={<FileArchive size={16} />}
            onClick={() => saveDiagnostics(game, profile)}
          >
            {t`Save diagnostics…`}
          </Button>
          <Box sx={{ flex: 1 }} />
          <Button disabled={busy} onClick={run}>
            {t`Run again`}
          </Button>
          <Button variant="contained" onClick={() => setOpen(false)}>
            {t`Close`}
          </Button>
        </DialogActions>
      </Dialog>
    </SettingsSection>
  )
}

export { Diagnostics }
