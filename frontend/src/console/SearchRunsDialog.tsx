import { useLingui } from '@lingui/react/macro'
import {
  alpha,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  ListItemButton,
  Typography,
} from '@mui/material'
import { FileSearch } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import type {
  RunHit,
  RunSearch,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { SearchRuns } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { SearchField } from '../shell/SearchField.tsx'
import { MONO } from '../theme/theme.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useOutcomeLabel } from './outcome.ts'
import { useConsole } from './store.ts'

const SEARCH_DEBOUNCE_MS = 250
const HIT_ALPHA = 0.35

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
            sx={{
              bgcolor: (theme) => alpha(theme.palette.warning.main, HIT_ALPHA),
              color: 'inherit',
            }}
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
        <SearchField
          label={t`Search run logs`}
          autoFocus={true}
          fullWidth={true}
          value={query}
          onChange={setQuery}
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
                  <ListItemButton
                    dense={true}
                    key={`${hit.runId}-${hit.lineNumber}`}
                    onClick={() => choose(hit)}
                    sx={{ gap: 1, borderRadius: 1, alignItems: 'flex-start' }}
                  >
                    <Typography
                      component="span"
                      sx={{ minWidth: 42, color: 'text.secondary', fontFamily: MONO }}
                    >
                      {t`L${hit.lineNumber}`}
                    </Typography>
                    <Typography
                      component="span"
                      sx={{
                        minWidth: 0,
                        whiteSpace: 'pre-wrap',
                        overflowWrap: 'anywhere',
                        fontFamily: MONO,
                      }}
                    >
                      <Highlight text={hit.line} query={query.trim()} />
                    </Typography>
                  </ListItemButton>
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
      <DialogActions>
        <Button onClick={onClose}>{t`Close`}</Button>
      </DialogActions>
    </Dialog>
  )
}
