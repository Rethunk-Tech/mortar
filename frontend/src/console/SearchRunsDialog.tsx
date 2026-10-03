import { useLingui } from '@lingui/react/macro'
import { Box, Dialog, DialogContent, DialogTitle, TextField, Typography } from '@mui/material'
import { FileSearch } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import type {
  RunHit,
  RunSearch,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { SearchRuns } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useOutcomeLabel } from './outcome.ts'
import { useConsole } from './store.ts'

const SEARCH_DEBOUNCE_MS = 250

function Highlight({ text, query }: { text: string; query: string }) {
  if (!query) {
    return <>{text}</>
  }
  const parts = text.split(new RegExp(`(${query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'ig'))
  return (
    <>
      {parts.map((part) =>
        part.toLowerCase() === query.toLowerCase() ? (
          <Box
            component="mark"
            key={part}
            sx={{ bgcolor: 'rgba(255,193,7,0.35)', color: 'inherit' }}
          >
            {part}
          </Box>
        ) : (
          part
        ),
      )}
    </>
  )
}

export function SearchRunsDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useLingui()
  const outcome = useOutcomeLabel()
  const shown = useConsole((s) => s.shown)
  const viewRun = useConsole((s) => s.viewRun)
  const jumpTo = useConsole((s) => s.jumpTo)
  const [query, setQuery] = useState('')
  const [result, setResult] = useState<RunSearch>({ hits: [], truncated: false })
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!open) {
      return
    }
    const trimmed = query.trim()
    if (!trimmed) {
      setResult({ hits: [], truncated: false })
      return
    }
    setLoading(true)
    const timer = globalThis.setTimeout(() => {
      SearchRuns(shown.game, shown.profile, trimmed)
        .then((next) => setResult(next ?? { hits: [], truncated: false }))
        .catch(reportUnexpected)
        .finally(() => setLoading(false))
    }, SEARCH_DEBOUNCE_MS)
    return () => globalThis.clearTimeout(timer)
  }, [open, query, shown.game, shown.profile])

  const groups = useMemo(() => {
    const grouped = new Map<string, RunHit[]>()
    for (const hit of result.hits ?? []) {
      const hits = grouped.get(hit.runId) ?? []
      hits.push(hit)
      grouped.set(hit.runId, hits)
    }
    return grouped
  }, [result.hits])

  const choose = (hit: RunHit) => {
    viewRun(shown.game, shown.profile, hit.runId)
    jumpTo(hit.lineNumber - 1)
    onClose()
  }

  return (
    <Dialog open={open} onClose={onClose} fullWidth={true} maxWidth="md">
      <DialogTitle>{t`Search all runs`}</DialogTitle>
      <DialogContent>
        <TextField
          autoFocus={true}
          fullWidth={true}
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder={t`Search run logs`}
          size="small"
          sx={{ mb: 1.5 }}
        />
        {loading ? <LoadingRow>{t`Searching…`}</LoadingRow> : null}
        {!loading && groups.size === 0 ? (
          <EmptyState
            icon={<FileSearch />}
            title={query.trim() ? t`No matches` : t`Search every run`}
            compact={true}
          >
            {query.trim()
              ? t`No stored run contains that text.`
              : t`Enter text to search stored runs.`}
          </EmptyState>
        ) : null}
        {!loading && groups.size > 0 ? (
          <>
            {Array.from(groups, ([runId, hits]) => (
              <Box key={runId} sx={{ mb: 1.5 }}>
                <Typography
                  sx={{ fontSize: 13, fontWeight: 700, color: 'text.secondary', mb: 0.5 }}
                >
                  {hits[0]
                    ? t`${formatWhen(hits[0].started, { withTime: true })} · ${outcome(hits[0].outcome)}`
                    : null}
                </Typography>
                {hits.map((hit) => (
                  <Box
                    component="button"
                    key={`${hit.runId}-${hit.lineNumber}`}
                    onClick={() => choose(hit)}
                    sx={{
                      display: 'flex',
                      width: '100%',
                      gap: 1,
                      border: 0,
                      borderRadius: 1,
                      px: 1,
                      py: 0.5,
                      color: 'inherit',
                      bgcolor: 'transparent',
                      textAlign: 'left',
                      cursor: 'pointer',
                      '&:hover': { bgcolor: 'var(--mortar-hairline-muted)' },
                    }}
                  >
                    <Typography
                      sx={{ minWidth: 42, color: 'text.secondary', fontFamily: 'monospace' }}
                    >
                      {t`L${hit.lineNumber}`}
                    </Typography>
                    <Typography
                      sx={{
                        minWidth: 0,
                        whiteSpace: 'pre-wrap',
                        overflowWrap: 'anywhere',
                        fontFamily: 'monospace',
                      }}
                    >
                      <Highlight text={hit.line} query={query.trim()} />
                    </Typography>
                  </Box>
                ))}
              </Box>
            ))}
            {result.truncated ? (
              <Typography color="text.secondary" sx={{ fontSize: 13 }}>
                {t`Showing the first 500 matches.`}
              </Typography>
            ) : null}
          </>
        ) : null}
      </DialogContent>
    </Dialog>
  )
}
