import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, MenuItem, Select, Typography } from '@mui/material'
import type {
  Entry,
  Mod,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useMods } from './store.ts'

const fieldSize = 13

// The channels a source offers: Nexus's file categories, GitHub's prereleases, and the release types CurseForge and
// Modrinth publish. A source without a channel gets none.
function channelOptions(i18n: I18n, kind: string) {
  switch (kind) {
    case 'nexus':
      return [
        { value: 'main', label: i18n._(msg`Main only`) },
        { value: 'optional', label: i18n._(msg`Optional files`) },
        { value: 'beta', label: i18n._(msg`Beta / prerelease`) },
      ]
    case 'github':
      return [
        { value: 'main', label: i18n._(msg`Releases only`) },
        { value: 'beta', label: i18n._(msg`Include prereleases`) },
      ]
    case 'curseforge':
    case 'modrinth':
      return [
        { value: 'main', label: i18n._(msg`Releases only`) },
        { value: 'optional', label: i18n._(msg`Releases and betas`) },
        { value: 'beta', label: i18n._(msg`Releases, betas and alphas`) },
      ]
    default:
      return []
  }
}

function ModUpdateControls({ mod, entry }: { mod: Mod; entry: Entry | undefined }) {
  const { t, i18n } = useLingui()
  const setUpdateChannel = useMods((s) => s.setUpdateChannel)
  const channel = entry?.updateChannel || 'main'
  const options = channelOptions(i18n, entry?.source.kind ?? '')
  if (options.length === 0) {
    return null
  }
  return (
    <Box>
      <Typography
        sx={{ fontSize: fieldSize, color: 'text.secondary' }}
      >{t`Update channel`}</Typography>
      <Select
        size="small"
        fullWidth={true}
        value={channel}
        inputProps={{ 'aria-label': t`Update channel` }}
        onChange={(ev) => setUpdateChannel(mod, String(ev.target.value)).catch(reportUnexpected)}
      >
        {options.map((opt) => (
          <MenuItem key={opt.value} value={opt.value}>
            {opt.label}
          </MenuItem>
        ))}
      </Select>
    </Box>
  )
}

export { ModUpdateControls }
