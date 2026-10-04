import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { alpha } from '@mui/material/styles'

const WARN_FILL = 0.1
const WARN_LINE = 0.4

import type { ShownInfo } from './logic.ts'

const SHOWN_NAMES = 30

const nameChip = {
  px: 1.25,
  py: 0.5,
  borderRadius: '4px',
  bgcolor: 'var(--mortar-card-hover)',
  fontSize: 13,
  whiteSpace: 'nowrap',
} as const

// What a share holds, by source, and what it leaves out and why. `notIn` names where the left-out mods are missing from.
export function SharedMods({ info, notIn }: { info: ShownInfo; notIn: string }) {
  const { t } = useLingui()
  const source = (id: string, n: number) => (id === 'github' ? t`${n} GitHub` : t`${n} Nexus`)
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
  const names = info.groups.flatMap((g) =>
    g.mods.map((name) => ({ key: `${g.source}-${name}`, name })),
  )
  const shown = names.slice(0, SHOWN_NAMES)
  return (
    <>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.25, p: '16px 24px' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.25, flexWrap: 'wrap' }}>
          <Typography sx={{ fontSize: 15, fontWeight: 600 }}>
            {plural(info.count, { one: '# mod included', other: '# mods included' })}
          </Typography>
          {info.groups.map((g) => (
            <Box
              key={g.source}
              component="span"
              sx={{
                px: 1,
                py: '2px',
                borderRadius: '10px',
                bgcolor: 'var(--mortar-hairline-muted)',
                fontSize: 12,
              }}
            >
              {source(g.source, g.count)}
            </Box>
          ))}
        </Box>
        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75 }}>
          {shown.map((m) => (
            <Box key={m.key} component="span" sx={nameChip}>
              {m.name}
            </Box>
          ))}
          {names.length > shown.length ? (
            <Box component="span" sx={{ px: 1.25, py: 0.5, fontSize: 13, color: 'text.secondary' }}>
              {t`and ${names.length - shown.length} more`}
            </Box>
          ) : null}
        </Box>
      </Box>
      {info.leftOut.length > 0 ? (
        <Box
          sx={{
            display: 'flex',
            flexDirection: 'column',
            gap: 1,
            m: '0 24px 18px',
            p: '12px 14px',
            bgcolor: (th) => alpha(th.palette.warning.main, WARN_FILL),
            border: '1px solid',
            borderColor: (th) => alpha(th.palette.warning.main, WARN_LINE),
            borderRadius: '6px',
          }}
        >
          <Typography sx={{ fontSize: 14, fontWeight: 600 }}>{notIn}</Typography>
          {info.leftOut.map((o) => (
            <Box
              key={`${o.name}-${o.reason}`}
              sx={{ display: 'flex', alignItems: 'center', gap: 1, fontSize: 13 }}
            >
              <Box
                component="span"
                sx={{ px: 1, py: '2px', borderRadius: '4px', bgcolor: 'var(--mortar-overlay-30)' }}
              >
                {o.name}
              </Box>
              <Box component="span" sx={{ color: 'text.secondary' }}>
                {reason(o.reason)}
              </Box>
            </Box>
          ))}
        </Box>
      ) : null}
    </>
  )
}
