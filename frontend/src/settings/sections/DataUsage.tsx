import { useLingui } from '@lingui/react/macro'
import { Box, Button, LinearProgress } from '@mui/material'
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

const MIN_KEPT = 1
const MAX_KEPT = 50
// Every usage row reserves the same action slot, so sizes line up whether or not a row has a button.
const ACTION_WIDTH = 168

function SizeRow({ label, size, action }: { label: string; size: number; action?: ReactNode }) {
  return (
    <SettingRow label={label}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
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
  return (
    <SettingsSection title={t`Usage`}>
      {games.map((g) => (
        <SizeRow key={g.game} label={g.name || g.game} size={g.size} />
      ))}
      <SizeRow label={t`Store`} size={usage.store} action={button(t`Clean up…`, onCleanUp)} />
      <SizeRow label={t`Cache`} size={usage.cache} action={button(t`Clear…`, onClearCache)} />
      <SizeRow
        label={t`Trash`}
        size={usage.trash}
        action={button(t`Deleted profiles…`, onDeletedProfiles)}
      />
      <SizeRow label={t`Save backups`} size={usage.backups} />
      {usage.sharedSavedKnown ? (
        <SizeRow label={t`Space saved by sharing files`} size={usage.sharedSaved} />
      ) : null}
      <SizeRow label={t`Total`} size={usage.total} />
    </SettingsSection>
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
