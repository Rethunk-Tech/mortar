import { useLingui } from '@lingui/react/macro'
import { Box, Button, IconButton, Tooltip, Typography } from '@mui/material'
import { RotateCcw, X } from 'lucide-react'
import type { ReactNode } from 'react'
import type { Item } from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/models.ts'
import {
  Cancel,
  Choose,
  Confirm,
  Dismiss,
  OpenPage,
  Retry,
  RetryFailed,
  Skip,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/service.ts'
import { QueueNeedsMerge } from '../install/QueueNeedsMerge.tsx'
import { QueueNeedsRoot } from '../install/QueueNeedsRoot.tsx'
import { accent } from '../mods/paper.ts'
import { LetterTile } from '../mods/parts.tsx'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { Fold } from './QueueFold.tsx'
import { downloadedKb, isActive, megabytes, megabytesPerSecond, profileOf } from './totals.ts'

const BLUE = '#2b8bda'
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

const names = (items: Item[]) =>
  items
    .slice(0, NAME_MAX)
    .map((i) => i.name || i.fileName)
    .join(', ')

const tile = (i: Item) => ({
  uniqueId: i.repo || String(i.modId),
  name: i.name || i.repo || String(i.modId),
  picture: i.picture ?? '',
})

// The item's name with the profile it installs into under it; the sheet is too narrow to fit both on one line.
function Title({ item, size }: { item: Item; size: number }) {
  const { t } = useLingui()
  const profile = useProfiles((s) => profileOf(item, s.game?.id, s.profiles))
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', minWidth: 0 }}>
      <Typography noWrap={true} sx={{ fontSize: size, fontWeight: 600 }}>
        {item.name || item.fileName}
      </Typography>
      {profile === null ? null : (
        <Typography noWrap={true} sx={{ fontSize: 12, color: 'text.secondary' }}>
          {profile ? t`into ${profile}` : t`into a deleted profile`}
        </Typography>
      )}
    </Box>
  )
}

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
        <Title item={item} size={14} />
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

// A card for an item that waits for the user, with its own actions on the right and any choices under the text.
function Callout({
  item,
  label,
  text,
  actions,
  children,
}: {
  item: Item
  label: string
  text: string
  actions: ReactNode
  children?: ReactNode
}) {
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: '10px',
        p: '14px',
        bgcolor: accent.fill,
        border: '1px solid',
        borderColor: 'primary.main',
        borderRadius: '8px',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
        <LetterTile mod={tile(item)} size={44} />
        <Box sx={{ flexGrow: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
          <Typography
            sx={{
              fontSize: 12,
              fontWeight: 700,
              letterSpacing: '0.06em',
              textTransform: 'uppercase',
              color: 'primary.main',
            }}
          >
            {label}
          </Typography>
          <Title item={item} size={16} />
        </Box>
        <Box sx={{ flexShrink: 0 }}>{actions}</Box>
      </Box>
      <Typography sx={{ fontSize: 13, lineHeight: 1.45 }}>{text}</Typography>
      {children}
    </Box>
  )
}

function Click({ item }: { item: Item }) {
  const { t } = useLingui()
  const [pending, run] = usePending()
  return (
    <Callout
      item={item}
      label={t`Needs your click`}
      text={t`Press Mod Manager Download on Nexus. Mortar picks it up and opens the next page.`}
      actions={
        <Box sx={{ display: 'flex', gap: 1 }}>
          <SkipButton item={item} />
          <Button
            variant="contained"
            disabled={pending}
            onClick={() => run(() => OpenPage(item.id))}
            sx={{ whiteSpace: 'nowrap' }}
          >
            {t`Open download page`}
          </Button>
        </Box>
      }
    />
  )
}

function SkipButton({ item }: { item: Item }) {
  const { t } = useLingui()
  return (
    <Button
      variant="outlined"
      color="inherit"
      onClick={() => Skip(item.id).catch(reportUnexpected)}
      sx={{ whiteSpace: 'nowrap' }}
    >
      {t`Skip`}
    </Button>
  )
}

function Choice({ item }: { item: Item }) {
  const { t } = useLingui()
  const [pending, run] = usePending()
  return (
    <Callout
      item={item}
      label={t`Choose a file`}
      text={t`${item.repo} ${item.tag} has several archives. Which one should Mortar install?`}
      actions={<SkipButton item={item} />}
    >
      {(item.assets ?? []).map((asset) => (
        <Button
          key={asset}
          variant="outlined"
          disabled={pending}
          onClick={() => run(() => Choose(item.id, asset))}
          sx={{ justifyContent: 'flex-start', textTransform: 'none', overflowWrap: 'anywhere' }}
        >
          {asset}
        </Button>
      ))}
    </Callout>
  )
}

function Confirmation({ item }: { item: Item }) {
  const { t } = useLingui()
  const [pending, run] = usePending()
  return (
    <Callout
      item={item}
      label={t`Check this download`}
      text={t`This download's mod is not known to come from ${item.repo}.`}
      actions={
        <>
          <SkipButton item={item} />
          <Button
            variant="contained"
            disabled={pending}
            onClick={() => run(() => Confirm(item.id))}
            sx={{ whiteSpace: 'nowrap' }}
          >
            {t`Install anyway`}
          </Button>
        </>
      }
    />
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
              <Tooltip title={t`Dismiss ${i.name}`}>
                <IconButton
                  aria-label={t`Dismiss ${i.name}`}
                  onClick={() => Dismiss(i.id).catch(reportUnexpected)}
                >
                  <X size={14} />
                </IconButton>
              </Tooltip>
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
          <Title item={item} size={14} />
          <Typography sx={{ ...detail, flexShrink: 0 }}>{text}</Typography>
        </Box>
        <Box sx={{ height: 4, borderRadius: '2px', bgcolor: 'rgba(255,255,255,0.1)' }}>
          <Box sx={{ width: `${item.progress}%`, height: 4, borderRadius: '2px', bgcolor: BLUE }} />
        </Box>
      </Box>
      {downloading ? (
        <Tooltip title={t`Cancel ${item.name}`}>
          <IconButton
            aria-label={t`Cancel ${item.name}`}
            onClick={() => Cancel(item.id).catch(reportUnexpected)}
          >
            <X size={14} />
          </IconButton>
        </Tooltip>
      ) : null}
    </Box>
  )
}

export function Body({ items }: { items: Item[] }) {
  const { t } = useLingui()
  const click = items.filter((i) => i.state === 'waiting-click')
  const choose = items.filter((i) => i.state === 'needs-choice')
  const confirm = items.filter((i) => i.state === 'needs-confirm')
  const needsRoot = items.some((i) => i.state === 'needs-root')
  const needsMerge = items.some((i) => i.state === 'needs-merge')
  const failed = items.filter((i) => i.state === 'failed')
  const active = items.filter(isActive)
  const next = items.filter((i) => i.state === 'queued')
  const done = items.filter((i) => i.state === 'done')
  const dropped = items.filter((i) => i.state === 'skipped' || i.state === 'cancelled')
  if (
    click.length +
      choose.length +
      confirm.length +
      (needsRoot ? 1 : 0) +
      (needsMerge ? 1 : 0) +
      failed.length +
      active.length +
      next.length +
      done.length +
      dropped.length ===
    0
  ) {
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
      {choose.map((i) => (
        <Choice key={i.id} item={i} />
      ))}
      {confirm.map((i) => (
        <Confirmation key={i.id} item={i} />
      ))}
      <QueueNeedsRoot items={items} />
      <QueueNeedsMerge items={items} />
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
                <Tooltip title={t`Skip ${i.name}`}>
                  <IconButton
                    aria-label={t`Skip ${i.name}`}
                    onClick={() => Skip(i.id).catch(reportUnexpected)}
                  >
                    <X size={14} />
                  </IconButton>
                </Tooltip>
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
            <Row
              key={i.id}
              item={i}
              sx={{ px: '10px' }}
              sub={
                i.unverified ? (
                  <Typography sx={{ ...detail, color: 'text.secondary' }}>
                    {t`Could not verify its source`}
                  </Typography>
                ) : null
              }
              actions={
                <Tooltip title={t`Dismiss ${i.name}`}>
                  <IconButton
                    aria-label={t`Dismiss ${i.name}`}
                    onClick={() => Dismiss(i.id).catch(reportUnexpected)}
                  >
                    <X size={14} />
                  </IconButton>
                </Tooltip>
              }
            />
          ))}
        </Fold>
      ) : null}
      {dropped.length > 0 ? (
        <Fold bg="rgba(55,55,65,0.6)" line={t`Skipped (${dropped.length}): ${names(dropped)}`}>
          {dropped.map((i) => (
            <Row
              key={i.id}
              item={i}
              sx={{ bgcolor: 'rgba(55,55,65,0.6)' }}
              sub={
                <Typography sx={{ ...detail, color: 'text.secondary' }}>
                  {i.state === 'cancelled' ? t`Cancelled` : t`Skipped`}
                </Typography>
              }
              actions={
                <Tooltip title={t`Dismiss ${i.name}`}>
                  <IconButton
                    aria-label={t`Dismiss ${i.name}`}
                    onClick={() => Dismiss(i.id).catch(reportUnexpected)}
                  >
                    <X size={14} />
                  </IconButton>
                </Tooltip>
              }
            />
          ))}
        </Fold>
      ) : null}
    </>
  )
}
