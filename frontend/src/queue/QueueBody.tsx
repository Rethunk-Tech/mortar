import { useLingui } from '@lingui/react/macro'
import { Box, Button, LinearProgress, Typography } from '@mui/material'
import { alpha, useTheme } from '@mui/material/styles'
import { Download, RotateCcw, X } from 'lucide-react'
import type { Item } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/models.ts'
import {
  Cancel,
  Choose,
  Confirm,
  ConfirmLarge,
  Dismiss,
  InstallAnyway,
  OpenPage,
  Retry,
  RetryFailed,
  Skip,
  SkipAll,
  SkipProfile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/service.ts'
import { WINDOWS } from '../brand/productNames.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { formatBytes, formatKb } from '../i18n/bytes.ts'
import { listNames } from '../i18n/list.ts'
import { QueueNeedsMerge } from '../install/QueueNeedsMerge.tsx'
import { QueueNeedsRoot } from '../install/QueueNeedsRoot.tsx'
import { LetterTile } from '../mods/LetterTile.tsx'
import { openSettings } from '../nav/store.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { space } from '../theme/density.ts'
import { useOverride } from '../toasts/avOverride.ts'
import { errorDetails } from '../toasts/errorKind.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { Callout, Title } from './Callout.tsx'
import { Fold } from './QueueFold.tsx'
import { QueueNeedsFomod } from './QueueNeedsFomod.tsx'
import { MoveButtons, NowButton, PauseButton, Paused, Row, SectionTitle } from './QueueRows.tsx'
import { detail, ROW } from './rowStyle.ts'
import { displayName, downloadedKb, isActive, tile } from './totals.ts'

const DONE_FILL = 0.08
const FAIL_FILL = 0.1
const FAIL_LINE = 0.35

const NAME_MAX = 3

const names = (items: Item[]) =>
  listNames(
    items.map((i) => displayName(i)),
    NAME_MAX,
  )

function DismissButton({ item }: { item: Item }) {
  const { t } = useLingui()
  return (
    <TipIconButton
      label={t`Dismiss ${{ name: item.name }}`}
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
      text={t`Press Mod Manager Download on ${{ source: sourceLabel('nexus') }}. Mortar picks it up and opens the next page.`}
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
      text={t`${{ repo: item.repo }} ${{ tag: item.tag }} has several archives. Which one should Mortar install?`}
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
      text={t`This download's mod is not known to come from ${{ repo: item.repo }}.`}
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

function LargeWait({ item }: { item: Item }) {
  const { t } = useLingui()
  const [pending, run] = usePending()
  const size = formatKb(item.sizeKb)
  return (
    <Callout
      item={item}
      label={t`Very large download`}
      text={t`This download is ${size}, which is unusually large for a mod.`}
      actions={
        <>
          <Button disabled={pending} autoFocus={true} onClick={() => run(() => Cancel(item.id))}>
            {t`Cancel`}
          </Button>
          <Button
            variant="contained"
            color="warning"
            disabled={pending}
            onClick={() => run(() => ConfirmLarge(item.id))}
          >
            {t`Continue anyway`}
          </Button>
        </>
      }
    />
  )
}

function Confirmations({ items }: { items: Item[] }) {
  return items.map((i) =>
    i.state === 'confirm-large' ? (
      <LargeWait key={i.id} item={i} />
    ) : (
      <Confirmation key={i.id} item={i} />
    ),
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
        {t`Failed (${{ count: items.length }})`}
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
              <FailureText item={i} />
            </Typography>
          }
          actions={
            <>
              {i.detection && !i.detection.removed ? <InstallAnywayButton item={i} /> : null}
              {i.errorKind === 'disk' ? (
                <Button size="small" onClick={() => openSettings('storage')}>
                  {t`Open storage settings`}
                </Button>
              ) : null}
              <Button
                size="small"
                startIcon={<RotateCcw size={14} />}
                onClick={() => Retry(i.id).catch(reportUnexpected)}
              >
                {i.partial ? t`Resume` : t`Retry`}
              </Button>
              <DismissButton item={i} />
            </>
          }
        />
      ))}
    </>
  )
}

// Why a download failed: what the antivirus flagged, or that Windows removed the file itself, else the error.
function FailureText({ item }: { item: Item }) {
  const { t } = useLingui()
  const det = item.detection
  if (!det) {
    return errorMessage(item.error)
  }
  if (det.removed) {
    return t`${{ os: WINDOWS }} removed the file as malware without saying which. If you trust it, restore it from ${{ os: WINDOWS }} Security's protection history.`
  }
  return t`Flagged by ${{ scanner: det.scanner }}: ${{ name: det.name }}${{ where: det.file ? ` in ${det.file}` : '' }}`
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
    ? [
        t`${{ done: formatKb(downloadedKb(item)) }} of ${{ total: formatKb(item.sizeKb) }}`,
        t`${{ rate: formatBytes(item.speed) }}/s`,
      ].join(' · ')
    : t`Installing`
  return (
    <Box sx={{ ...ROW, bgcolor: 'var(--mortar-raised)' }}>
      <LetterTile mod={tile(item)} size={36} />
      <Box
        sx={{ flexGrow: 1, minWidth: 0, display: 'flex', flexDirection: 'column', gap: space.gap }}
      >
        <Box sx={{ display: 'flex', justifyContent: 'space-between', gap: space.gap }}>
          <Title item={item} size={14} />
          <Typography sx={{ ...detail, flexShrink: 0 }}>{text}</Typography>
        </Box>
        <LinearProgress
          variant="determinate"
          color="info"
          value={item.progress}
          aria-label={t`Download progress for ${{ name: item.name }}`}
          sx={{
            height: 4,
            borderRadius: '2px',
            bgcolor: 'var(--mortar-hairline)',
            '& .MuiLinearProgress-bar': { borderRadius: '2px' },
          }}
        />
      </Box>
      {downloading ? (
        <>
          <PauseButton item={item} />
          <TipIconButton
            label={t`Cancel ${{ name: item.name }}`}
            onClick={() => Cancel(item.id).catch(reportUnexpected)}
          >
            <X size={14} />
          </TipIconButton>
        </>
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

function Skipped({ items }: { items: Item[] }) {
  const { t } = useLingui()
  const droppedReason = (i: Item) => {
    if (i.state === 'cancelled') {
      return t`Cancelled`
    }
    if (i.error === 'Already the newest file') {
      return t`Already the newest file`
    }
    return i.error ? errorMessage(i.error) : t`Skipped`
  }
  if (items.length === 0) {
    return null
  }
  return (
    <Fold
      bg="var(--mortar-raised-60)"
      line={t`Skipped (${{ count: items.length }}): ${{ names: names(items) }}`}
    >
      {items.map((i) => (
        <Row
          key={i.id}
          item={i}
          sx={{ bgcolor: 'var(--mortar-raised-60)' }}
          sub={
            <Typography sx={{ ...detail, color: 'text.secondary' }}>{droppedReason(i)}</Typography>
          }
          actions={<DismissButton item={i} />}
        />
      ))}
    </Fold>
  )
}

export function Body({ items, onBrowse }: { items: Item[]; onBrowse: () => void }) {
  const { t } = useLingui()
  const theme = useTheme()
  const doneBg = alpha(theme.palette.success.main, DONE_FILL)
  const click = items.filter((i) => i.state === 'waiting-click')
  const choose = items.filter((i) => i.state === 'needs-choice')
  const confirm = items.filter((i) => i.state === 'needs-confirm' || i.state === 'confirm-large')
  const needsRoot = items.some((i) => i.state === 'needs-root')
  const needsMerge = items.some((i) => i.state === 'needs-merge')
  const fomod = items.filter((i) => i.state === 'needs-fomod')
  const failed = items.filter((i) => i.state === 'failed')
  const active = items.filter(isActive)
  const next = items.filter((i) => i.state === 'queued')
  const paused = items.filter((i) => i.state === 'paused')
  const done = items.filter((i) => i.state === 'done')
  const dropped = items.filter((i) => i.state === 'skipped' || i.state === 'cancelled')
  if (
    click.length +
      choose.length +
      confirm.length +
      (needsRoot ? 1 : 0) +
      (needsMerge ? 1 : 0) +
      fomod.length +
      failed.length +
      active.length +
      paused.length +
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
      <Confirmations items={confirm} />
      <QueueNeedsRoot items={items} />
      <QueueNeedsMerge items={items} />
      <QueueNeedsFomod items={fomod} />
      <Failed items={failed} />
      {active.length > 0 ? (
        <SectionTitle>{t`In progress (${{ count: active.length }})`}</SectionTitle>
      ) : null}
      {active.map((i) => (
        <Active key={i.id} item={i} />
      ))}
      <Paused items={paused} />
      {next.length > 0 ? (
        <Fold
          bg="var(--mortar-raised-60)"
          line={t`Up next: ${{ names: names(next) }}`}
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
                <>
                  <MoveButtons item={i} />
                  <NowButton item={i} />
                  <PauseButton item={i} />
                  <TipIconButton
                    label={t`Skip ${{ name: i.name }}`}
                    onClick={() => Skip(i.id).catch(reportUnexpected)}
                  >
                    <X size={14} />
                  </TipIconButton>
                </>
              }
            />
          ))}
        </Fold>
      ) : null}
      {done.length > 0 ? (
        <Fold
          bg={doneBg}
          color={theme.palette.success.light}
          line={t`Done (${{ count: done.length }}): ${{ names: names(done) }}`}
        >
          {done.map((i) => (
            <Row
              key={i.id}
              item={i}
              sx={{ px: space.gap }}
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
      <Skipped items={dropped} />
    </>
  )
}
