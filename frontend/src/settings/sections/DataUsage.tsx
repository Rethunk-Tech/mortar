import { useLingui } from '@lingui/react/macro'
import { Box, Button, LinearProgress, Skeleton, Tooltip, useTheme } from '@mui/material'
import { FolderOpen } from 'lucide-react'
import { type ReactNode, useEffect, useState } from 'react'
import type { Usage as DiskUse } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/models.ts'
import { DataLocation } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/service.ts'
import { OpenDataFolder } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { formatBytes } from '../../i18n/bytes.ts'
import { cmpText } from '../../mods/cmpText.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { Searchable, SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { MoveDataButton } from './DataMove.tsx'
import { nowrap } from './dataStyles.ts'
import { type SegmentId, storageSegments } from './storageSegments.ts'

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
const SKELETON_WIDTH = 64
const LEGEND_SKELETON = 40
const SEGMENT_ORDER: SegmentId[] = ['profiles', 'store', 'cache', 'backups', 'trash', 'other']
const filled = { ...nowrap, bgcolor: 'var(--mortar-raised)', boxShadow: 'none' } as const

function useSegmentColors() {
  return SEGMENT_COLORS[useTheme().palette.mode]
}

function SizeRow({ label, size }: { label: string; size: number | null }) {
  return (
    <SettingRow label={label}>
      <Box sx={{ fontSize: 15, fontWeight: 600, fontVariantNumeric: 'tabular-nums' }}>
        {size === null ? <Skeleton width={SKELETON_WIDTH} /> : formatBytes(size)}
      </Box>
    </SettingRow>
  )
}

// While measuring, the legend keeps the entries the last measurement in this session had, so it does not
// shrink when the numbers arrive; the first measurement starts from the two that are never empty in use.
let lastShown: SegmentId[] = ['profiles', 'store']

function legendIds(sizes: Record<SegmentId, number> | null): SegmentId[] {
  if (sizes === null) {
    return lastShown
  }
  lastShown = SEGMENT_ORDER.filter((id) => sizes[id] > 0)
  return lastShown
}

function Legend({
  sizes,
  labels,
}: {
  sizes: Record<SegmentId, number> | null
  labels: Record<SegmentId, string>
}) {
  const colors = useSegmentColors()
  return (
    <Box sx={{ display: 'flex', flexWrap: 'wrap', columnGap: 2, rowGap: 0.75, pt: 1.5 }}>
      {legendIds(sizes).map((id) => (
        <Box
          key={id}
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: 0.5,
            fontSize: 13,
            whiteSpace: 'nowrap',
          }}
        >
          <Box
            aria-hidden={true}
            sx={{
              width: DOT,
              height: DOT,
              borderRadius: '50%',
              bgcolor: colors[id],
              flexShrink: 0,
            }}
          />
          <Box component="span" sx={{ fontWeight: 600 }}>
            {labels[id]}
          </Box>
          <Box
            component="span"
            sx={{ color: 'text.secondary', fontVariantNumeric: 'tabular-nums' }}
          >
            {sizes ? formatBytes(sizes[id]) : <Skeleton width={LEGEND_SKELETON} />}
          </Box>
        </Box>
      ))}
    </Box>
  )
}

function StorageBar({
  usage,
  labels,
  bytes,
  actions,
}: {
  usage: DiskUse | null
  labels: Record<SegmentId, string>
  bytes: number
  actions: ReactNode
}) {
  const { t } = useLingui()
  const colors = useSegmentColors()
  const segs = usage ? storageSegments(usage).filter((s) => s.size > 0) : []
  const sizes = usage
    ? (Object.fromEntries(storageSegments(usage).map((s) => [s.id, s.size])) as Record<
        SegmentId,
        number
      >)
    : null
  const total = Math.max(1, usage?.total ?? 1)
  return (
    <Box sx={{ px: 2.5, py: 2 }}>
      <Box sx={{ display: 'flex', alignItems: 'baseline', gap: 2, pb: 1.5 }}>
        <Box sx={{ flex: 1, fontSize: 16 }}>
          {usage ? t`${formatBytes(usage.total)} used` : t`Measuring… ${formatBytes(bytes)}`}
        </Box>
        {usage?.sharedSavedKnown && usage.sharedSaved > 0 ? (
          <Box sx={{ fontSize: 14, color: 'text.secondary' }}>
            {t`${formatBytes(usage.sharedSaved)} saved by sharing files`}
          </Box>
        ) : null}
      </Box>
      {usage ? (
        <Box
          role="img"
          aria-label={segs.map((s) => `${labels[s.id]} ${formatBytes(s.size)}`).join(', ')}
          sx={{ display: 'flex', gap: `${BAR_GAP}px`, height: BAR_HEIGHT }}
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
      ) : (
        <LinearProgress sx={{ height: BAR_HEIGHT, borderRadius: `${END_RADIUS}px` }} />
      )}
      <Legend sizes={sizes} labels={labels} />
      <Box sx={{ display: 'flex', gap: 1, pt: 2, flexWrap: 'wrap' }}>{actions}</Box>
    </Box>
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
  const games = usage
    ? [...(usage.games ?? [])].sort((a, b) => b.size - a.size || cmpText(a.name, b.name))
    : null
  const labels: Record<SegmentId, string> = {
    profiles: t`Profiles`,
    store: t`Store`,
    cache: t`Cache`,
    backups: t`Save backups`,
    trash: t`Recently deleted`,
    other: t`Other`,
  }
  const actions = (
    <>
      <Button variant="outlined" color="inherit" onClick={onCleanUp} sx={filled}>
        {t`Clean up…`}
      </Button>
      <Button variant="outlined" color="inherit" onClick={onClearCache} sx={filled}>
        {t`Clear cache…`}
      </Button>
      <Button variant="outlined" color="inherit" onClick={onDeletedProfiles} sx={filled}>
        {t`Open recently deleted`}
      </Button>
    </>
  )
  return (
    <>
      <SettingsSection title={t`Usage`}>
        <Searchable
          terms={`${t`Usage`} ${t`Storage`} ${t`Disk space`} ${Object.values(labels).join(' ')} ${t`Clean up…`} ${t`Clear cache…`} ${t`Open recently deleted`}`}
        >
          <StorageBar usage={usage} labels={labels} bytes={bytes} actions={actions} />
        </Searchable>
      </SettingsSection>
      <SettingsSection title={t`By game`}>
        {games === null ? (
          <SizeRow label={t`Measuring…`} size={null} />
        ) : (
          games.map((g) => (
            <SizeRow
              key={g.game}
              label={g.game === '' ? t`Shared by every game` : g.name || g.game}
              size={g.size}
            />
          ))
        )}
      </SettingsSection>
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
  const [portableDir, setPortableDir] = useState('')
  useEffect(() => {
    DataLocation()
      .then((loc) => setPortableDir(loc.portable ? loc.dir : ''))
      .catch(reportUnexpected)
  }, [])
  const portable = portableDir !== ''
  let description = usage ? usage.path : t`Measuring…`
  if (portable) {
    description = t`Portable copy: data is kept in ${portableDir}`
  }
  return (
    <SettingsSection title={t`Location`}>
      <SettingRow label={t`Mortar's data`} description={description}>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Button
            variant="outlined"
            startIcon={<FolderOpen size={16} />}
            onClick={() => OpenDataFolder().catch(reportUnexpected)}
          >
            {t`Open folder`}
          </Button>
          <MoveDataButton
            onPicked={onPicked}
            disabledReason={portable ? t`A portable copy keeps its data beside the program.` : ''}
          />
        </Box>
      </SettingRow>
    </SettingsSection>
  )
}
