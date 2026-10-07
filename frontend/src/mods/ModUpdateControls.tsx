import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, MenuItem, Select, Tooltip, Typography } from '@mui/material'
import type {
  Entry,
  Mod,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useMods } from './store.ts'

const fieldSize = 13

function channelOptions(i18n: I18n) {
  return [
    { value: 'main', label: i18n._(msg`Main only`) },
    { value: 'optional', label: i18n._(msg`Optional files`) },
    { value: 'beta', label: i18n._(msg`Beta / prerelease`) },
  ]
}

function ModUpdateControls({ mod, entry }: { mod: Mod; entry: Entry | undefined }) {
  const { t, i18n } = useLingui()
  const setPinned = useMods((s) => s.setPinned)
  const setUpdateChannel = useMods((s) => s.setUpdateChannel)
  const channel = entry?.updateChannel || 'main'
  return (
    <>
      <Tooltip
        title={
          entry?.pinned
            ? t`Pinned: Mortar offers no update for this mod until you unpin it.`
            : t`Mortar stops offering updates for this mod and keeps the version you have.`
        }
      >
        <Button
          variant="outlined"
          onClick={() => setPinned(mod, !entry?.pinned).catch(reportUnexpected)}
        >
          {entry?.pinned ? t`Unpin version` : t`Pin this version`}
        </Button>
      </Tooltip>
      {/* The channels are Nexus file categories, which no other source has. */}
      {entry?.source.kind === 'nexus' ? (
        <Box>
          <Typography
            sx={{ fontSize: fieldSize, color: 'text.secondary' }}
          >{t`Update channel`}</Typography>
          <Select
            size="small"
            fullWidth={true}
            value={channel}
            inputProps={{ 'aria-label': t`Update channel` }}
            onChange={(ev) =>
              setUpdateChannel(mod, String(ev.target.value)).catch(reportUnexpected)
            }
          >
            {channelOptions(i18n).map((opt) => (
              <MenuItem key={opt.value} value={opt.value}>
                {opt.label}
              </MenuItem>
            ))}
          </Select>
        </Box>
      ) : null}
    </>
  )
}

export { ModUpdateControls }
