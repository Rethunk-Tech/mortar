import { useLingui } from '@lingui/react/macro'
import { Box, Button, Drawer, IconButton, Typography } from '@mui/material'
import { History as HistoryIcon, List, X } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import {
  ClearFinished,
  Pause,
  Resume,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/service.ts'
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
import { clockTime, megabytes, totals } from './totals.ts'

const WIDTH = 500
const GREEN = '#0cdf64'
const BLUE = '#2b8bda'
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
  const counts = t`${sum.done} done · ${sum.active} in progress · ${sum.failed} failed · ${sum.left} left`
  const line = sum.sizeKb > 0 ? t`${counts} · ${megabytes(sum.sizeKb)} MB` : counts
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
        <Button
          variant="outlined"
          color="inherit"
          startIcon={view === 'history' ? <List size={14} /> : <HistoryIcon size={14} />}
          onClick={() => onView(view === 'history' ? 'queue' : 'history')}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {view === 'history' ? t`Queue` : t`History`}
        </Button>
        {finished.length > 0 && view === 'queue' ? (
          <Button
            variant="outlined"
            color="inherit"
            onClick={() => ClearFinished().catch(reportUnexpected)}
            sx={{ whiteSpace: 'nowrap' }}
          >
            {t`Clear finished`}
          </Button>
        ) : null}
        {idle ? null : (
          <Button
            variant="outlined"
            color="inherit"
            onClick={() => (paused ? Resume() : Pause()).catch(reportUnexpected)}
            sx={{ whiteSpace: 'nowrap' }}
          >
            {paused ? t`Resume` : t`Pause all`}
          </Button>
        )}
        <IconButton aria-label={t`Close downloads`} onClick={onClose}>
          <X size={16} />
        </IconButton>
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
          bgcolor: 'rgba(255,255,255,0.1)',
        }}
      >
        <Box sx={{ width: `${sum.doneShare}%`, bgcolor: GREEN }} />
        <Box sx={{ width: `${sum.activeShare}%`, bgcolor: BLUE }} />
        <Box sx={{ width: `${sum.failedShare}%`, bgcolor: RED }} />
      </Box>
      {limitedUntil > 0 ? (
        <Typography sx={{ mx: 2.5, mt: 1, fontSize: 13, color: 'warning.main' }}>
          {t`Nexus or GitHub has limited requests for now. Downloads go on at ${clockTime(limitedUntil)}.`}
        </Typography>
      ) : null}
    </>
  )
}

export function QueueSheet() {
  const open = useQueue((s) => s.open)
  const setOpen = useQueue((s) => s.setOpen)
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
            bgcolor: 'rgb(34,34,42)',
            borderLeft: '1px solid rgba(255,255,255,0.12)',
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
