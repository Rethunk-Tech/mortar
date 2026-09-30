import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { heading } from '../mods/paper.ts'
import type { ShownInfo } from './logic.ts'

const SEPARATOR = ' · '

// What a share holds, by source, and what it leaves out and why.
export function SharedMods({ info }: { info: ShownInfo }) {
  const { t } = useLingui()
  const source = (id: string) => (id === 'github' ? t`GitHub` : t`Nexus Mods`)
  const reason = (id: string) => {
    switch (id) {
      case 'local':
        return t`Added from an archive, so it only exists on this computer.`
      case 'off':
        return t`Switched off in this profile.`
      default:
        return t`Its source is not known.`
    }
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
      {info.groups.map((g) => (
        <Box key={g.source}>
          <Typography sx={{ ...heading, mb: 0.75 }}>
            {t`${source(g.source)} (${g.mods.length})`}
          </Typography>
          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75 }}>
            {g.mods.map((name) => (
              <Box
                key={name}
                sx={{
                  px: 1.25,
                  py: 0.5,
                  borderRadius: '12px',
                  bgcolor: 'rgba(255,255,255,0.1)',
                  fontSize: 13,
                }}
              >
                {name}
              </Box>
            ))}
          </Box>
        </Box>
      ))}
      {info.leftOut.length > 0 ? (
        <Box>
          <Typography sx={{ ...heading, mb: 0.75 }}>{t`Left out`}</Typography>
          {info.leftOut.map((o) => (
            <Typography
              key={`${o.name}-${o.reason}`}
              sx={{ fontSize: 13, color: 'text.secondary' }}
            >
              <b>{o.name}</b>
              {SEPARATOR}
              {reason(o.reason)}
            </Typography>
          ))}
        </Box>
      ) : null}
    </Box>
  )
}
