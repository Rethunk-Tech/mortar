import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Drawer, IconButton, Tooltip, Typography } from '@mui/material'
import {
  History as HistoryIcon,
  List,
  ListX,
  Pause as PauseIcon,
  Play as PlayIcon,
  X,
} from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import {
  ClearFinished,
  Pause,
  Resume,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/service.ts'
import { formatKb } from '../i18n/bytes.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { reportUnexpected } from '../toasts/report.ts'
import {
  emptyFilters,
  type HistoryEntry,
  type HistoryFilters,
  loadHistory,
  makeGen,
} from './history.ts'
import { Body } from './QueueBody.tsx'
import { HistoryList } from './QueueHistory.tsx'
import { useQueue } from './store.ts'
import { clockTime, parallelDownloads, totals } from './totals.ts'

const UNIX_MS_PER_SECOND = 1000
const WIDTH = 500
const RED = '#ff6b5f'

function Header({
  onClose,
  view,
  onView,
}: {
  onClose: () => void
  view: 'queue' | 'history'
  onView: (view: 'queue' | 'history') => void
}) {
  const { t } = useLingui()
  const { items, paused, limitedUntil } = useQueue((s) => s.state)
  const sum = totals(items)
  const { downloading, waiting } = parallelDownloads(items)
  const counts = t`${sum.done} done · ${sum.active} in progress · ${sum.failed} failed · ${sum.left} left`
  const parallel =
    downloading > 0 || waiting > 0
      ? t`${plural(downloading, { one: '# downloading', other: '# downloading' })} · ${plural(waiting, { one: '# waiting', other: '# waiting' })}`
      : null
  const summary = parallel ? t`${parallel} · ${counts}` : counts
  const line = sum.sizeKb > 0 ? t`${summary} · ${formatKb(sum.sizeKb)}` : summary
  const idle = !paused && sum.active === 0 && sum.left === 0
  const finished = items.filter((i) => ['done', 'failed', 'skipped', 'cancelled'].includes(i.state))
  return (
    <>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, pt: 2, pr: 1, pb: 1.25, pl: 2.5 }}>
        <Box sx={{ flexGrow: 1, minWidth: 0 }}>
          <Typography component="h2" sx={{ fontSize: 18, fontWeight: 700 }}>
            {view === 'history' ? t`History` : t`Downloads`}
          </Typography>
          <Typography noWrap={true} sx={{ fontSize: 13 }}>
            {line}
          </Typography>
        </Box>
        <Tooltip title={view === 'history' ? t`Queue` : t`History`}>
          <IconButton
            aria-label={view === 'history' ? t`Queue` : t`History`}
            onClick={() => onView(view === 'history' ? 'queue' : 'history')}
          >
            {view === 'history' ? <List size={16} /> : <HistoryIcon size={16} />}
          </IconButton>
        </Tooltip>
        {finished.length > 0 && view === 'queue' ? (
          <Tooltip title={t`Clear finished`}>
            <IconButton
              aria-label={t`Clear finished`}
              onClick={() => ClearFinished().catch(reportUnexpected)}
            >
              <ListX size={16} />
            </IconButton>
          </Tooltip>
        ) : null}
        {idle ? null : (
          <Tooltip title={paused ? t`Resume` : t`Pause all`}>
            <IconButton
              aria-label={paused ? t`Resume` : t`Pause all`}
              onClick={() => (paused ? Resume() : Pause()).catch(reportUnexpected)}
            >
              {paused ? <PlayIcon size={16} /> : <PauseIcon size={16} />}
            </IconButton>
          </Tooltip>
        )}
        <Tooltip title={t`Close downloads`}>
          <IconButton aria-label={t`Close downloads`} onClick={onClose}>
            <X size={16} />
          </IconButton>
        </Tooltip>
      </Box>
      <Box
        role="progressbar"
        aria-label={t`Download progress`}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Math.round(sum.doneShare)}
        sx={{
          display: 'flex',
          height: 6,
          mx: 2.5,
          borderRadius: '3px',
          overflow: 'hidden',
          bgcolor: 'var(--mortar-hairline)',
        }}
      >
        <Box sx={{ width: `${sum.doneShare}%`, bgcolor: 'success.main' }} />
        <Box sx={{ width: `${sum.activeShare}%`, bgcolor: 'info.main' }} />
        <Box sx={{ width: `${sum.failedShare}%`, bgcolor: RED }} />
      </Box>
      {limitedUntil > 0 ? (
        <Typography
          title={clockTime(limitedUntil)}
          sx={{ mx: 2.5, mt: 1, fontSize: 13, color: 'warning.main' }}
        >
          {t`Downloads resume ${formatWhen(limitedUntil * UNIX_MS_PER_SECOND)}.`}
        </Typography>
      ) : null}
    </>
  )
}

export function QueueSheet() {
  const open = useQueue((s) => s.open)
  const setOpen = useQueue((s) => s.setOpen)
  const historyBatchId = useQueue((s) => s.historyBatchId)
  const items = useQueue((s) => s.state.items)
  const [view, setView] = useState<'queue' | 'history'>('queue')
  const [history, setHistory] = useState<HistoryEntry[]>([])
  const [filters, setFilters] = useState<HistoryFilters>(emptyFilters)
  const gen = useRef(makeGen())
  const close = () => {
    setView('queue')
    setOpen(false)
  }
  useEffect(() => {
    if (open && historyBatchId) {
      setView('history')
      setFilters({ outcome: 'failed', profileId: '', batchId: historyBatchId })
      useQueue.getState().consumeHistoryBatch()
    }
  }, [open, historyBatchId])
  useEffect(() => {
    if (!open || view !== 'history') {
      return
    }
    const id = gen.current.stamp()
    loadHistory()
      .then((rows) => {
        if (gen.current.is(id)) {
          setHistory(rows)
        }
      })
      .catch(reportUnexpected)
    return () => {
      gen.current.drop()
    }
  }, [open, view])
  return (
    <Drawer
      anchor="right"
      open={open}
      onClose={close}
      sx={{ top: 'var(--title-bar)' }}
      slotProps={{
        paper: {
          role: 'dialog',
          sx: {
            width: WIDTH,
            maxWidth: '100%',
            top: 'var(--title-bar)',
            height: 'calc(100% - var(--title-bar))',
            bgcolor: 'var(--mortar-panel-solid)',
            borderLeft: '1px solid var(--mortar-hairline-12)',
          },
        },
        backdrop: { sx: { top: 'var(--title-bar)' } },
      }}
    >
      <Header onClose={close} view={view} onView={setView} />
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          gap: '12px',
          p: '14px 20px',
          overflowY: 'auto',
        }}
      >
        {view === 'history' ? (
          <HistoryList
            entries={history}
            filters={filters}
            onFilters={setFilters}
            onCleared={() => {
              gen.current.drop()
              setHistory([])
            }}
          />
        ) : (
          <Body items={items} />
        )}
      </Box>
    </Drawer>
  )
}
