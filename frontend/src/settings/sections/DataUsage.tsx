import { useLingui } from '@lingui/react/macro'
import { Box, Button, LinearProgress, Tooltip, useTheme } from '@mui/material'
import { FolderOpen } from 'lucide-react'
import type { ReactNode } from 'react'
import type { Usage as DiskUse } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/datasvc/models.ts'
import {
  OpenDataFolder,
  SetBackupsKept,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { formatBytes } from '../../i18n/bytes.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { PrefNumber } from '../PrefControls.tsx'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'
import { MoveDataButton } from './DataMove.tsx'
import { nowrap } from './dataStyles.ts'
import { type SegmentId, storageSegments } from './storageSegments.ts'

const MIN_KEPT = 1
const MAX_KEPT = 50
// Every usage row reserves the same action slot, so sizes line up whether or not a row has a button.
const ACTION_WIDTH = 168

// Categorical slots validated for both themes (CVD and contrast); "other" stays neutral so it reads as remainder.
const SEGMENT_COLORS: Record<'light' | 'dark', Record<SegmentId, string>> = {
  light: {
    profiles: '#2a78d6',
    store: '#eb6834',
    cache: '#1baf7a',
    backups: '#eda100',
    trash: '#e87ba4',
    other: '#9e9e9e',
  },
  dark: {
    profiles: '#3987e5',
    store: '#d95926',
    cache: '#199e70',
    backups: '#c98500',
    trash: '#d55181',
    other: '#757575',
  },
}
const DOT = 10
const BAR_HEIGHT = 12
const BAR_GAP = 2
const END_RADIUS = 4
const MIN_SEGMENT = 4
const PERCENT = 100

function useSegmentColors() {
  return SEGMENT_COLORS[useTheme().palette.mode]
}

function SizeRow({
  label,
  size,
  action,
  color,
}: {
  label: string
  size: number
  action?: ReactNode
  color?: string
}) {
  return (
    <SettingRow label={label}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
        {color ? (
          <Box
            aria-hidden={true}
            sx={{ width: DOT, height: DOT, borderRadius: '50%', bgcolor: color, flexShrink: 0 }}
          />
        ) : null}
        <Box sx={{ fontSize: 14, fontVariantNumeric: 'tabular-nums', textAlign: 'right' }}>
          {formatBytes(size)}
        </Box>
        <Box sx={{ width: ACTION_WIDTH, display: 'flex', justifyContent: 'flex-end' }}>
          {action}
        </Box>
      </Box>
    </SettingRow>
  )
}

function StorageBar({ usage, labels }: { usage: DiskUse; labels: Record<SegmentId, string> }) {
  const colors = useSegmentColors()
  const segs = storageSegments(usage).filter((s) => s.size > 0)
  const total = Math.max(1, usage.total)
  return (
    <Box
      role="img"
      aria-label={segs.map((s) => `${labels[s.id]} ${formatBytes(s.size)}`).join(', ')}
      sx={{ display: 'flex', gap: `${BAR_GAP}px`, height: BAR_HEIGHT, px: 2, py: 2 }}
    >
      {segs.map((s, i) => (
        <Tooltip
          key={s.id}
          title={`${labels[s.id]} · ${formatBytes(s.size)} · ${Math.round((s.size / total) * PERCENT)}%`}
        >
          <Box
            sx={{
              flexGrow: s.size,
              flexBasis: 0,
              minWidth: MIN_SEGMENT,
              bgcolor: colors[s.id],
              borderTopLeftRadius: i === 0 ? END_RADIUS : 0,
              borderBottomLeftRadius: i === 0 ? END_RADIUS : 0,
              borderTopRightRadius: i === segs.length - 1 ? END_RADIUS : 0,
              borderBottomRightRadius: i === segs.length - 1 ? END_RADIUS : 0,
            }}
          />
        </Tooltip>
      ))}
    </Box>
  )
}

export function BackupsKept() {
  const { t } = useLingui()
  const kept = useSettings((s) => s.backupsKept)
  return (
    <SettingRow
      label={t`Save backups kept`}
      description={t`Saves are zipped before mods update; older backups beyond this many are deleted. ${MIN_KEPT} to ${MAX_KEPT}.`}
    >
      <PrefNumber value={kept} min={MIN_KEPT} max={MAX_KEPT} onCommit={SetBackupsKept} />
    </SettingRow>
  )
}

export function UsageRows({
  usage,
  bytes,
  onCleanUp,
  onClearCache,
  onDeletedProfiles,
}: {
  usage: DiskUse | null
  bytes: number
  onCleanUp: () => void
  onClearCache: () => void
  onDeletedProfiles: () => void
}) {
  const { t } = useLingui()
  const colors = useSegmentColors()
  if (!usage) {
    return (
      <SettingsSection title={t`Usage`}>
        <SettingRow label={t`Measuring…`} description={formatBytes(bytes)}>
          <LinearProgress sx={{ width: 160 }} />
        </SettingRow>
      </SettingsSection>
    )
  }
  const games = [...(usage.games ?? [])].sort(
    (a, b) => b.size - a.size || a.name.localeCompare(b.name),
  )
  const button = (label: string, onClick: () => void) => (
    <Button variant="outlined" onClick={onClick} sx={nowrap}>
      {label}
    </Button>
  )
  const labels: Record<SegmentId, string> = {
    profiles: t`Profiles`,
    store: t`Store`,
    cache: t`Cache`,
    backups: t`Save backups`,
    trash: t`Trash`,
    other: t`Other`,
  }
  const sizes = Object.fromEntries(storageSegments(usage).map((s) => [s.id, s.size])) as Record<
    SegmentId,
    number
  >
  const actions: Partial<Record<SegmentId, ReactNode>> = {
    store: button(t`Clean up…`, onCleanUp),
    cache: button(t`Clear…`, onClearCache),
    trash: button(t`Deleted profiles…`, onDeletedProfiles),
  }
  const order: SegmentId[] = ['profiles', 'store', 'cache', 'backups', 'trash', 'other']
  return (
    <>
      <SettingsSection title={t`Usage`}>
        <StorageBar usage={usage} labels={labels} />
        {order
          .filter((id) => id !== 'other' || sizes.other > 0)
          .map((id) => (
            <SizeRow
              key={id}
              label={labels[id]}
              size={sizes[id]}
              color={colors[id]}
              action={actions[id]}
            />
          ))}
        {usage.sharedSavedKnown ? (
          <SizeRow label={t`Space saved by sharing files`} size={usage.sharedSaved} />
        ) : null}
        <SizeRow label={t`Total`} size={usage.total} />
      </SettingsSection>
      {games.length > 0 ? (
        <SettingsSection title={t`By game`}>
          {games.map((g) => (
            <SizeRow key={g.game} label={g.name || g.game} size={g.size} />
          ))}
        </SettingsSection>
      ) : null}
    </>
  )
}

export function Location({
  usage,
  onPicked,
}: {
  usage: DiskUse | null
  onPicked: (dest: string) => void
}) {
  const { t } = useLingui()
  return (
    <SettingsSection title={t`Location`}>
      <SettingRow label={t`Mortar's data`} description={usage ? usage.path : t`Measuring…`}>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Button
            variant="outlined"
            startIcon={<FolderOpen size={16} />}
            onClick={() => OpenDataFolder().catch(reportUnexpected)}
            sx={nowrap}
          >
            {t`Open folder`}
          </Button>
          <MoveDataButton onPicked={onPicked} />
        </Box>
      </SettingRow>
    </SettingsSection>
  )
}
