import { useLingui } from '@lingui/react/macro'
import { Box, Button, Chip, Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material'
import { useEffect, useState } from 'react'
import type { Changelog } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexus/models.ts'
import { ReleaseChangelog } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { ErrorRetry } from '../shell/ErrorRetry.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { inlineError } from '../toasts/report.ts'
import { changelogNoteIsRisky } from './changelogRange.ts'
import { useNexusEntry } from './nexusDetails.ts'
import { isNewer, isSameVersion } from './nexusFormat.ts'
import { ChangelogText } from './updateReview/Changes.tsx'

type Loaded = { logs: Changelog[] } | { error: ReturnType<typeof inlineError> } | null

// GitHub releases come from the cached releases list; Nexus versions are already in the mod's cached details.
function useReleases(open: boolean, githubRepo: string) {
  const [state, setState] = useState<Loaded>(null)
  const [attempt, setAttempt] = useState(0)
  useEffect(() => {
    if (!open || githubRepo === '' || attempt < 0) {
      return
    }
    let live = true
    setState(null)
    ReleaseChangelog(githubRepo).then(
      (logs) => live && setState({ logs: logs ?? [] }),
      (e: unknown) => live && setState({ error: inlineError(e) }),
    )
    return () => {
      live = false
    }
  }, [open, githubRepo, attempt])
  return { state, retry: () => setAttempt((n) => n + 1) }
}

function Version({ entry, installed }: { entry: Changelog; installed: string }) {
  const { t } = useLingui()
  const current = isSameVersion(entry.version, installed)
  const fresh = isNewer(entry.version, installed)
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: 0.25,
        pl: 1.5,
        py: 0.5,
        borderLeft: '3px solid',
        borderLeftColor: fresh ? 'primary.main' : 'transparent',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <Box sx={{ fontWeight: 700, fontSize: 14 }}>
          {entry.date ? `${entry.version} · ${entry.date}` : entry.version}
        </Box>
        {current ? <Chip size="small" color="success" label={t`Installed`} /> : null}
        {fresh ? <Chip size="small" color="primary" label={t`New`} /> : null}
      </Box>
      {entry.body ? <ChangelogText text={entry.body} /> : null}
      {(entry.notes ?? []).map((n) => (
        <ChangelogText key={n} text={`• ${n}`} risky={changelogNoteIsRisky(n)} />
      ))}
    </Box>
  )
}

export function ChangelogDialog({
  open,
  onClose,
  name,
  installed,
  nexusId,
  githubRepo,
}: {
  open: boolean
  onClose: () => void
  name: string
  installed: string
  nexusId: number
  githubRepo: string
}) {
  const { t } = useLingui()
  const nexusLogs = useNexusEntry(nexusId)?.details?.changelogs
  const { state, retry } = useReleases(open, nexusId > 0 ? '' : githubRepo)
  let logs: Changelog[] | null = null
  if (nexusId > 0) {
    logs = nexusLogs ?? []
  } else if (state !== null && 'logs' in state) {
    ;({ logs } = state)
  }
  let body = <LoadingRow>{t`Reading the changelog…`}</LoadingRow>
  if (state !== null && 'error' in state) {
    body = <ErrorRetry error={state.error} onRetry={retry} />
  } else if (logs !== null && logs.length === 0) {
    body = (
      <EmptyState compact={true} icon={null} title={t`No changelog published`}>
        {t`The author has not listed any versions here.`}
      </EmptyState>
    )
  } else if (logs !== null) {
    body = (
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
        {logs.map((entry) => (
          <Version key={entry.version} entry={entry} installed={installed} />
        ))}
      </Box>
    )
  }
  return (
    <Dialog open={open} onClose={onClose} fullWidth={true} maxWidth="sm">
      <DialogTitle>{t`Changelog: ${name}`}</DialogTitle>
      <DialogContent>{body}</DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t`Close`}</Button>
      </DialogActions>
    </Dialog>
  )
}
