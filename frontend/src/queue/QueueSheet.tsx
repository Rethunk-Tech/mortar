import { useLingui } from '@lingui/react/macro'
import { Box, Drawer, Typography } from '@mui/material'
import { clockWhen } from '@rethunk/react-load-hooks'
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
  Dismiss,
  Pause,
  Resume,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/service.ts'
import { useTab } from '../game/tab.ts'
import { formatKb } from '../i18n/bytes.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { space } from '../theme/density.ts'
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
import { isClearableFinished, parallelDownloads, totals } from './totals.ts'

const UNIX_MS_PER_SECOND = 1000
const WIDTH = 500

function Header({
  onClose,
  view,
  onView,
}: {
  onClose: () => void
  view: 'queue' | 'history'
  onView: (view: 'queue' | 'history') => void
}) {
  const { t, i18n } = useLingui()
  const { items, paused, limitedUntil } = useQueue((s) => s.state)
  const sum = totals(items)
  const { downloading, waiting } = parallelDownloads(items)
  const installing = items.filter((i) => i.state === 'installing').length
  const pausedItems = items.filter((i) => i.state === 'paused').length
  const n = { downloading, installing, waiting, paused: pausedItems }
  // Each state once, none that is empty: the sheet is too narrow for more.
  const summary = [
    downloading > 0 && t`${{ downloading: n.downloading }} downloading`,
    installing > 0 && t`${{ installing: n.installing }} installing`,
    waiting > 0 && t`${{ count: n.waiting }} waiting`,
    pausedItems > 0 && t`${{ paused: n.paused }} paused`,
    sum.failed > 0 && t`${{ failed: sum.failed }} failed`,
    sum.done > 0 && t`${{ done: sum.done }} done`,
  ]
    .filter(Boolean)
    .join(' · ')
  const line = sum.sizeKb > 0 ? `${summary} · ${formatKb(sum.sizeKb)}` : summary
  const idle = !paused && sum.active === 0 && sum.left === 0
  const finished = items.filter((i) => isClearableFinished(i.state))
  return (
    <>
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: space.gap,
          pt: space.pad,
          pr: space.gap,
          pb: space.gap,
          pl: space.pad,
        }}
      >
        <Box sx={{ flexGrow: 1, minWidth: 0 }}>
          <Typography component="h2" sx={{ fontSize: 18, fontWeight: 700 }}>
            {view === 'history' ? t`History` : t`Downloads`}
          </Typography>
          <Typography noWrap={true} title={line} sx={{ fontSize: 'base.fontSize' }}>
            {line}
          </Typography>
        </Box>
        <TipIconButton
          label={view === 'history' ? t`Queue` : t`History`}
          onClick={() => onView(view === 'history' ? 'queue' : 'history')}
        >
          {view === 'history' ? <List size={16} /> : <HistoryIcon size={16} />}
        </TipIconButton>
        {finished.length > 0 && view === 'queue' ? (
          <TipIconButton
            label={t`Clear finished`}
            onClick={() =>
              Promise.all(finished.map((item) => Dismiss(item.id))).catch(reportUnexpected)
            }
          >
            <ListX size={16} />
          </TipIconButton>
        ) : null}
        {idle ? null : (
          <TipIconButton
            label={paused ? t`Resume` : t`Pause all`}
            onClick={() => (paused ? Resume() : Pause()).catch(reportUnexpected)}
          >
            {paused ? <PlayIcon size={16} /> : <PauseIcon size={16} />}
          </TipIconButton>
        )}
        <TipIconButton label={t`Close downloads`} onClick={onClose}>
          <X size={16} />
        </TipIconButton>
      </Box>
      {items.length === 0 || view === 'history' ? null : (
        <Box
          role="progressbar"
          aria-label={t`Download progress`}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={Math.round(sum.doneShare)}
          sx={{
            display: 'flex',
            height: 6,
            mx: space.pad,
            borderRadius: '3px',
            overflow: 'hidden',
            bgcolor: 'var(--mortar-hairline)',
          }}
        >
          <Box sx={{ width: `${sum.doneShare}%`, bgcolor: 'success.main' }} />
          <Box sx={{ width: `${sum.activeShare}%`, bgcolor: 'info.main' }} />
          <Box sx={{ width: `${sum.failedShare}%`, bgcolor: 'error.main' }} />
        </Box>
      )}
      {limitedUntil > 0 ? (
        <Typography
          title={clockWhen(limitedUntil * UNIX_MS_PER_SECOND, i18n.locale)}
          sx={{ mx: space.pad, mt: space.gap, fontSize: 'base.fontSize', color: 'warning.main' }}
        >
          {t`Downloads resume ${{ when: formatWhen(limitedUntil * UNIX_MS_PER_SECOND) }}.`}
        </Typography>
      ) : null}
    </>
  )
}

export function QueueSheet() {
  const { t } = useLingui()
  const open = useQueue((s) => s.open)
  const setOpen = useQueue((s) => s.setOpen)
  const historyBatchId = useQueue((s) => s.historyBatchId)
  const items = useQueue((s) => s.state.items)
  const [view, setView] = useState<'queue' | 'history'>('queue')
  // null until read, so opening History never flashes its empty message first.
  const [history, setHistory] = useState<HistoryEntry[] | null>(null)
  const [filters, setFilters] = useState<HistoryFilters>(emptyFilters)
  const gen = useRef(makeGen())
  // History gains a row whenever a download settles.
  const settled = items.filter((i) => i.state === 'failed' || isClearableFinished(i.state)).length
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
    if (!open || view !== 'history' || settled < 0) {
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
  }, [open, view, settled])
  return (
    <Drawer
      anchor="right"
      open={open}
      onClose={close}
      sx={{ top: 'var(--title-bar)' }}
      slotProps={{
        paper: {
          role: 'dialog',
          'aria-label': view === 'history' ? t`Download history` : t`Downloads`,
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
          gap: space.gap,
          p: space.pad,
          minHeight: 0,
          flex: 1,
          overflowY: view === 'history' ? 'hidden' : 'auto',
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
          <Body
            items={items}
            onBrowse={() => {
              close()
              useTab.getState().setTab('browse')
            }}
          />
        )}
      </Box>
    </Drawer>
  )
}
