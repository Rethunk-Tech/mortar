import { useLingui } from '@lingui/react/macro'
import { Box, Button, LinearProgress, Typography } from '@mui/material'
import { alpha, useTheme } from '@mui/material/styles'
import { listNames } from '../i18n/list.ts'

const DONE_FILL = 0.08
const FAIL_FILL = 0.1
const FAIL_LINE = 0.35

import { Download, RotateCcw, X } from 'lucide-react'
import type { ReactNode } from 'react'
import type { Item } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/models.ts'
import {
  Cancel,
  Choose,
  Confirm,
  Dismiss,
  InstallAnyway,
  OpenPage,
  Retry,
  RetryFailed,
  Skip,
  SkipAll,
  SkipProfile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/service.ts'
import { formatBytes, formatKb } from '../i18n/bytes.ts'
import { QueueNeedsMerge } from '../install/QueueNeedsMerge.tsx'
import { QueueNeedsRoot } from '../install/QueueNeedsRoot.tsx'
import { LetterTile } from '../mods/parts.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { space } from '../theme/density.ts'
import { useOverride } from '../toasts/avOverride.ts'
import { errorDetails } from '../toasts/errorKind.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { Callout, Title } from './Callout.tsx'
import { Fold } from './QueueFold.tsx'
import { displayName, downloadedKb, isActive, tile } from './totals.ts'

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
  listNames(
    items.map((i) => displayName(i)),
    NAME_MAX,
  )

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
        color: color ?? 'var(--mortar-ink-sec)',
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

function DismissButton({ item }: { item: Item }) {
  const { t } = useLingui()
  return (
    <TipIconButton
      label={t`Dismiss ${item.name}`}
      onClick={() => Dismiss(item.id).catch(reportUnexpected)}
    >
      <X size={14} />
    </TipIconButton>
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
        <Box sx={{ display: 'flex', gap: space.gap }}>
          <SkipButton item={item} />
          <Button
            variant="contained"
            disabled={pending}
            onClick={() => run(() => OpenPage(item.id))}
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
  const { error } = useTheme().palette
  const [pending, run] = usePending()
  if (items.length === 0) {
    return null
  }
  return (
    <>
      <SectionTitle
        color={error.light}
        action={
          <TipIconButton
            label={t`Retry failed`}
            color="error"
            disabled={pending}
            onClick={() => run(() => RetryFailed())}
          >
            <RotateCcw size={16} />
          </TipIconButton>
        }
      >
        {t`Failed (${items.length})`}
      </SectionTitle>
      {items.map((i) => (
        <Row
          key={i.id}
          item={i}
          sx={{
            bgcolor: alpha(error.main, FAIL_FILL),
            border: `1px solid ${alpha(error.main, FAIL_LINE)}`,
          }}
          sub={
            <Typography title={errorDetails(i.error)} sx={{ ...detail, color: error.light }}>
              {i.detection
                ? t`Flagged by ${i.detection.scanner}: ${i.detection.name}${i.detection.file ? ` in ${i.detection.file}` : ''}`
                : errorMessage(i.error)}
            </Typography>
          }
          actions={
            <>
              {i.detection ? <InstallAnywayButton item={i} /> : null}
              <Button
                size="small"
                startIcon={<RotateCcw size={14} />}
                onClick={() => Retry(i.id).catch(reportUnexpected)}
              >
                {t`Retry`}
              </Button>
              <DismissButton item={i} />
            </>
          }
        />
      ))}
    </>
  )
}

// The antivirus flagged this download; installing it anyway is the player's call, behind a confirm.
function InstallAnywayButton({ item }: { item: Item }) {
  const { t } = useLingui()
  const det = item.detection
  if (!det) {
    return null
  }
  return (
    <Button
      size="small"
      color="error"
      onClick={() =>
        useOverride.getState().ask({
          title: displayName(item),
          scanner: det.scanner,
          name: det.name,
          file: det.file,
          confirm: () => InstallAnyway(item.id),
        })
      }
    >
      {t`Install anyway`}
    </Button>
  )
}

function Active({ item }: { item: Item }) {
  const { t } = useLingui()
  const downloading = item.state === 'downloading'
  const text = downloading
    ? t`${formatKb(downloadedKb(item))} of ${formatKb(item.sizeKb)} · ${formatBytes(item.speed)}/s`
    : t`Installing`
  return (
    <Box sx={{ ...ROW, bgcolor: 'var(--mortar-raised)' }}>
      <LetterTile mod={tile(item)} size={36} />
      <Box sx={{ flexGrow: 1, minWidth: 0, display: 'flex', flexDirection: 'column', gap: '5px' }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', gap: space.gap }}>
          <Title item={item} size={14} />
          <Typography sx={{ ...detail, flexShrink: 0 }}>{text}</Typography>
        </Box>
        <LinearProgress
          variant="determinate"
          color="info"
          value={item.progress}
          aria-label={t`Download progress for ${item.name}`}
          sx={{
            height: 4,
            borderRadius: '2px',
            bgcolor: 'var(--mortar-hairline)',
            '& .MuiLinearProgress-bar': { borderRadius: '2px' },
          }}
        />
      </Box>
      {downloading ? (
        <TipIconButton
          label={t`Cancel ${item.name}`}
          onClick={() => Cancel(item.id).catch(reportUnexpected)}
        >
          <X size={14} />
        </TipIconButton>
      ) : null}
    </Box>
  )
}

function NextActions({ item }: { item: Item }) {
  const { t } = useLingui()
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
      <Button
        size="small"
        title={t`Skip every waiting download`}
        onClick={() => SkipAll().catch(reportUnexpected)}
      >
        {t`Skip all`}
      </Button>
      <Button
        size="small"
        title={t`Skip the waiting downloads of this profile`}
        onClick={() => SkipProfile(item.game, item.profileId).catch(reportUnexpected)}
      >
        {t`Skip this profile`}
      </Button>
    </Box>
  )
}

export function Body({ items, onBrowse }: { items: Item[]; onBrowse: () => void }) {
  const { t } = useLingui()
  const theme = useTheme()
  const doneBg = alpha(theme.palette.success.main, DONE_FILL)
  const click = items.filter((i) => i.state === 'waiting-click')
  const choose = items.filter((i) => i.state === 'needs-choice')
  const confirm = items.filter((i) => i.state === 'needs-confirm')
  const needsRoot = items.some((i) => i.state === 'needs-root')
  const needsMerge = items.some((i) => i.state === 'needs-merge')
  const failed = items.filter((i) => i.state === 'failed')
  const active = items.filter(isActive)
  const next = items.filter((i) => i.state === 'queued')
  const done = items.filter((i) => i.state === 'done')
  const droppedReason = (i: Item) => {
    if (i.state === 'cancelled') {
      return t`Cancelled`
    }
    return i.error === 'Already the newest file'
      ? t`Already the newest file`
      : i.error || t`Skipped`
  }
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
      <EmptyState
        icon={<Download />}
        title={t`Nothing downloading`}
        action={
          <Button variant="outlined" onClick={onBrowse}>
            {t`Browse mods`}
          </Button>
        }
      >
        {t`Updates, missing dependencies and links from mod sites land here.`}
      </EmptyState>
    )
  }
  const [firstProfile] = next
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
          bg="var(--mortar-raised-60)"
          line={t`Up next: ${names(next)}`}
          action={firstProfile ? <NextActions item={firstProfile} /> : null}
        >
          {next.map((i) => (
            <Row
              key={i.id}
              item={i}
              sx={{ bgcolor: 'var(--mortar-raised-60)' }}
              sub={
                <Typography sx={{ ...detail, color: 'text.secondary' }}>{i.fileName}</Typography>
              }
              actions={
                <TipIconButton
                  label={t`Skip ${i.name}`}
                  onClick={() => Skip(i.id).catch(reportUnexpected)}
                >
                  <X size={14} />
                </TipIconButton>
              }
            />
          ))}
        </Fold>
      ) : null}
      {done.length > 0 ? (
        <Fold
          bg={doneBg}
          color={theme.palette.success.light}
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
              actions={<DismissButton item={i} />}
            />
          ))}
        </Fold>
      ) : null}
      {dropped.length > 0 ? (
        <Fold bg="var(--mortar-raised-60)" line={t`Skipped (${dropped.length}): ${names(dropped)}`}>
          {dropped.map((i) => (
            <Row
              key={i.id}
              item={i}
              sx={{ bgcolor: 'var(--mortar-raised-60)' }}
              sub={
                <Typography sx={{ ...detail, color: 'text.secondary' }}>
                  {droppedReason(i)}
                </Typography>
              }
              actions={<DismissButton item={i} />}
            />
          ))}
        </Fold>
      ) : null}
    </>
  )
}
