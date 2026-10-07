import { useLingui } from '@lingui/react/macro'
import { Box, Button, Chip, Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material'
import type { Changelog } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexus/models.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { ErrorRetry } from '../shell/ErrorRetry.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { space } from '../theme/density.ts'
import { changelogNoteIsRisky } from './changelogRange.ts'
import { MarkdownView } from './MarkdownView.tsx'
import { useNexusEntry } from './nexusDetails.ts'
import { isNewer, isSameVersion } from './nexusFormat.ts'
import { useReleases } from './releases.ts'
import { ChangelogText } from './updateReview/Changes.tsx'

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
        pl: space.pad,
        py: 0.5,
        borderLeft: '3px solid',
        borderLeftColor: fresh ? 'primary.main' : 'transparent',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
        <Box sx={{ fontWeight: 700, fontSize: 14 }}>
          {entry.date ? `${entry.version} · ${entry.date}` : entry.version}
        </Box>
        {current ? <Chip size="small" color="success" label={t`Installed`} /> : null}
        {fresh ? <Chip size="small" color="primary" label={t`New`} /> : null}
      </Box>
      {entry.body ? <MarkdownView source={entry.body} /> : null}
      {(entry.notes ?? []).map((n) => (
        <ChangelogText key={n} text={`• ${n}`} risky={changelogNoteIsRisky(n)} />
      ))}
    </Box>
  )
}

/** The changelog's versions, newest first, each marked New or Installed against installed. */
export function ChangelogEntries({ logs, installed }: { logs: Changelog[]; installed: string }) {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}>
      {logs.map((entry) => (
        <Version key={entry.version} entry={entry} installed={installed} />
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
  markdown = '',
}: {
  open: boolean
  onClose: () => void
  name: string
  installed: string
  nexusId: number
  githubRepo: string
  // A changelog the site keeps as one Markdown page (a Thunderstore CHANGELOG), shown as it is.
  markdown?: string
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
    body = <ChangelogEntries logs={logs} installed={installed} />
  }
  if (markdown !== '') {
    body = <MarkdownView source={markdown} />
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
