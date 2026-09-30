import { useLingui } from '@lingui/react/macro'
import { Box, Button, Drawer, IconButton, Typography } from '@mui/material'
import { ChevronDown, ChevronUp, RotateCcw, X } from 'lucide-react'
import { type ReactNode, useState } from 'react'
import type { Item } from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/models.ts'
import {
  Cancel,
  OpenPage,
  Pause,
  Resume,
  Retry,
  RetryFailed,
  Skip,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/service.ts'
import { LetterTile } from '../mods/parts.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useQueue } from './store.ts'
import {
  clockTime,
  downloadedKb,
  isActive,
  megabytes,
  megabytesPerSecond,
  totals,
} from './totals.ts'

const WIDTH = 500
const GREEN = '#0cdf64'
const BLUE = '#2b8bda'
const RED = '#ff6b5f'
const ROW = {
  display: 'flex',
  alignItems: 'center',
  gap: '10px',
  height: 54,
  pl: '10px',
  pr: '6px',
  borderRadius: '6px',
} as const
const NAME_MAX = 3

const tile = (i: Item) => ({
  uniqueId: String(i.modId),
  name: i.name || String(i.modId),
  picture: '',
})

function SectionTitle({
  color,
  children,
  action,
}: {
  color?: string
  children: ReactNode
  action?: ReactNode
}) {
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        fontSize: 12,
        fontWeight: 700,
        letterSpacing: '0.06em',
        textTransform: 'uppercase',
        color: color ?? 'rgba(225,225,230,0.95)',
      }}
    >
      {children}
      {action}
    </Box>
  )
}

function Row({
  item,
  sub,
  actions,
  sx,
}: {
  item: Item
  sub: ReactNode
  actions: ReactNode
  sx?: object
}) {
  return (
    <Box sx={{ ...ROW, ...sx }}>
      <LetterTile mod={tile(item)} size={36} />
      <Box sx={{ flexGrow: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
        <Typography noWrap={true} sx={{ fontSize: 14, fontWeight: 600 }}>
          {item.name || item.fileName}
        </Typography>
        {sub}
      </Box>
      {actions}
    </Box>
  )
}

const detail = {
  fontSize: 12,
  whiteSpace: 'nowrap',
  overflow: 'hidden',
  textOverflow: 'ellipsis',
} as const

function Click({ item }: { item: Item }) {
  const { t } = useLingui()
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: '10px',
        p: '14px',
        bgcolor: 'rgba(214,177,122,0.14)',
        border: '1px solid',
        borderColor: 'primary.main',
        borderRadius: '8px',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
        <LetterTile mod={tile(item)} size={44} />
        <Box sx={{ flexGrow: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
          <Typography
            sx={{ fontSize: 12, fontWeight: 700, letterSpacing: '0.06em', color: 'primary.main' }}
          >
            {t`NEEDS YOUR CLICK`}
          </Typography>
          <Typography noWrap={true} sx={{ fontSize: 16, fontWeight: 600 }}>
            {item.name || item.fileName}
          </Typography>
        </Box>
        <Button
          variant="contained"
          onClick={() => OpenPage(item.id).catch(reportUnexpected)}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {t`Open download page`}
        </Button>
      </Box>
      <Typography sx={{ fontSize: 13, lineHeight: 1.45 }}>
        {t`Press Mod Manager Download on Nexus. Mortar picks it up and opens the next page.`}
      </Typography>
    </Box>
  )
}

function Failed({ items }: { items: Item[] }) {
  const { t } = useLingui()
  if (items.length === 0) {
    return null
  }
  return (
    <>
      <SectionTitle
        color="#ffb3ab"
        action={
          <Button
            size="small"
            color="error"
            variant="outlined"
            onClick={() => RetryFailed().catch(reportUnexpected)}
            sx={{ whiteSpace: 'nowrap' }}
          >
            {t`Retry failed`}
          </Button>
        }
      >
        {t`Failed (${items.length})`}
      </SectionTitle>
      {items.map((i) => (
        <Row
          key={i.id}
          item={i}
          sx={{ bgcolor: 'rgba(255,107,95,0.1)', border: '1px solid rgba(255,107,95,0.35)' }}
          sub={
            <Typography title={i.error} sx={{ ...detail, color: '#ffc4be' }}>
              {i.error}
            </Typography>
          }
          actions={
            <>
              <Button
                size="small"
                startIcon={<RotateCcw size={14} />}
                onClick={() => Retry(i.id).catch(reportUnexpected)}
                sx={{ whiteSpace: 'nowrap' }}
              >
                {t`Retry`}
              </Button>
              <IconButton
                aria-label={t`Skip ${i.name}`}
                onClick={() => Skip(i.id).catch(reportUnexpected)}
              >
                <X size={14} />
              </IconButton>
            </>
          }
        />
      ))}
    </>
  )
}

function Active({ item }: { item: Item }) {
  const { t } = useLingui()
  const downloading = item.state === 'downloading'
  const text = downloading
    ? t`${megabytes(downloadedKb(item))} of ${megabytes(item.sizeKb)} MB · ${megabytesPerSecond(item.speed)} MB/s`
    : t`Installing`
  return (
    <Box sx={{ ...ROW, bgcolor: 'rgba(55,55,65,0.9)' }}>
      <LetterTile mod={tile(item)} size={36} />
      <Box sx={{ flexGrow: 1, minWidth: 0, display: 'flex', flexDirection: 'column', gap: '5px' }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', gap: 1 }}>
          <Typography noWrap={true} sx={{ fontSize: 14, fontWeight: 600 }}>
            {item.name || item.fileName}
          </Typography>
          <Typography sx={{ ...detail, flexShrink: 0 }}>{text}</Typography>
        </Box>
        <Box sx={{ height: 4, borderRadius: '2px', bgcolor: 'rgba(255,255,255,0.1)' }}>
          <Box sx={{ width: `${item.progress}%`, height: 4, borderRadius: '2px', bgcolor: BLUE }} />
        </Box>
      </Box>
      {downloading ? (
        <IconButton
          aria-label={t`Cancel ${item.name}`}
          onClick={() => Cancel(item.id).catch(reportUnexpected)}
        >
          <X size={14} />
        </IconButton>
      ) : null}
    </Box>
  )
}

// One line that opens into the rows it stands for.
function Fold({
  line,
  color,
  bg,
  children,
}: {
  line: string
  color?: string
  bg: string
  children: ReactNode
}) {
  const [open, setOpen] = useState(false)
  return (
    <>
      <Box
        component="button"
        onClick={() => setOpen(!open)}
        aria-expanded={open}
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: 1,
          minHeight: 40,
          px: '10px',
          border: 0,
          borderRadius: '6px',
          bgcolor: bg,
          color: color ?? 'inherit',
          fontFamily: 'inherit',
          fontSize: 13,
          textAlign: 'left',
          cursor: 'pointer',
        }}
      >
        <Box
          component="span"
          sx={{ minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
        >
          {line}
        </Box>
        {open ? (
          <ChevronUp size={14} aria-hidden={true} />
        ) : (
          <ChevronDown size={14} aria-hidden={true} />
        )}
      </Box>
      {open ? children : null}
    </>
  )
}

const names = (items: Item[]) =>
  items
    .slice(0, NAME_MAX)
    .map((i) => i.name || i.fileName)
    .join(', ')

function Body({ items }: { items: Item[] }) {
  const { t } = useLingui()
  const click = items.filter((i) => i.state === 'waiting-click')
  const failed = items.filter((i) => i.state === 'failed')
  const active = items.filter(isActive)
  const next = items.filter((i) => i.state === 'queued')
  const done = items.filter((i) => i.state === 'done')
  if (click.length + failed.length + active.length + next.length + done.length === 0) {
    return (
      <Typography sx={{ fontSize: 14, color: 'text.secondary' }}>
        {t`Nothing is downloading. Updates, missing dependencies and links from Nexus land here.`}
      </Typography>
    )
  }
  const more = next.length - NAME_MAX
  return (
    <>
      {click.map((i) => (
        <Click key={i.id} item={i} />
      ))}
      <Failed items={failed} />
      {active.length > 0 ? <SectionTitle>{t`In progress (${active.length})`}</SectionTitle> : null}
      {active.map((i) => (
        <Active key={i.id} item={i} />
      ))}
      {next.length > 0 ? (
        <Fold
          bg="rgba(55,55,65,0.6)"
          line={more > 0 ? t`Up next: ${names(next)} and ${more} more` : t`Up next: ${names(next)}`}
        >
          {next.map((i) => (
            <Row
              key={i.id}
              item={i}
              sx={{ bgcolor: 'rgba(55,55,65,0.6)' }}
              sub={
                <Typography sx={{ ...detail, color: 'text.secondary' }}>{i.fileName}</Typography>
              }
              actions={
                <IconButton
                  aria-label={t`Skip ${i.name}`}
                  onClick={() => Skip(i.id).catch(reportUnexpected)}
                >
                  <X size={14} />
                </IconButton>
              }
            />
          ))}
        </Fold>
      ) : null}
      {done.length > 0 ? (
        <Fold
          bg="rgba(12,223,100,0.08)"
          color="#6ff5a8"
          line={t`Done (${done.length}): ${names(done)}`}
        >
          {done.map((i) => (
            <Typography key={i.id} sx={{ px: '10px', fontSize: 13 }}>
              {i.name || i.fileName}
            </Typography>
          ))}
        </Fold>
      ) : null}
    </>
  )
}

function Header({ onClose }: { onClose: () => void }) {
  const { t } = useLingui()
  const { items, paused, limitedUntil } = useQueue((s) => s.state)
  const sum = totals(items)
  const line = t`${sum.done} done · ${sum.active} in progress · ${sum.failed} failed · ${sum.left} left · ${megabytes(sum.sizeKb)} MB`
  return (
    <>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, pt: 2, pr: 1, pb: 1.25, pl: 2.5 }}>
        <Box sx={{ flexGrow: 1, minWidth: 0 }}>
          <Typography component="h2" sx={{ fontSize: 18, fontWeight: 700 }}>
            {t`Downloads`}
          </Typography>
          <Typography noWrap={true} sx={{ fontSize: 13 }}>
            {line}
          </Typography>
        </Box>
        <Button
          variant="outlined"
          color="inherit"
          onClick={() => (paused ? Resume() : Pause()).catch(reportUnexpected)}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {paused ? t`Resume` : t`Pause all`}
        </Button>
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
          {t`Nexus has limited requests for now. Downloads go on at ${clockTime(limitedUntil)}.`}
        </Typography>
      ) : null}
    </>
  )
}

export function QueueSheet() {
  const open = useQueue((s) => s.open)
  const setOpen = useQueue((s) => s.setOpen)
  const items = useQueue((s) => s.state.items)
  const close = () => setOpen(false)
  return (
    <Drawer
      anchor="right"
      open={open}
      onClose={close}
      sx={{ top: 'var(--title-bar)' }}
      slotProps={{
        paper: {
          sx: {
            width: WIDTH,
            maxWidth: '100%',
            top: 'var(--title-bar)',
            height: 'calc(100% - var(--title-bar))',
            bgcolor: 'rgba(34,34,42,0.98)',
            borderLeft: '1px solid rgba(255,255,255,0.12)',
          },
        },
        backdrop: { sx: { top: 'var(--title-bar)' } },
      }}
    >
      <Header onClose={close} />
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          gap: '12px',
          p: '14px 20px',
          overflowY: 'auto',
        }}
      >
        <Body items={items} />
      </Box>
    </Drawer>
  )
}
