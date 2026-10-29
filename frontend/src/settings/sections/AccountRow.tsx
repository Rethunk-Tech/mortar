import { Box } from '@mui/material'
import type { ReactNode } from 'react'
import { SourceLogo } from '../../brand/sources/SourceLogo.tsx'
import { sourceLabel } from '../../brand/sources/sourceLabel.ts'
import { space } from '../../theme/density.ts'
import { SettingRow } from '../SettingsSection.tsx'

export type AccountTone = 'ok' | 'bad' | 'none'

const DOT_COLOR: Record<AccountTone, string> = {
  ok: 'success.main',
  bad: 'error.main',
  none: 'text.disabled',
}

const LOGO_SIZE = 24

/** The first row of a service's block: its logo and name, one line saying who you are (or how the connection
 * stands, for a service with no account) behind a status dot, and the one action that fits. */
export function AccountRow({
  source,
  status,
  tone,
  action,
}: {
  source: string
  // Plain text, so a search finds the row by what it says.
  status: string
  tone: AccountTone
  action?: ReactNode
}) {
  const name = sourceLabel(source)
  return (
    <SettingRow label={`${name}: ${status}`} hideTitle={true} block={true}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.pad }}>
        <Box
          aria-hidden={true}
          sx={{
            width: LOGO_SIZE,
            height: LOGO_SIZE,
            flexShrink: 0,
            display: 'grid',
            placeItems: 'center',
          }}
        >
          <SourceLogo id={source} size={LOGO_SIZE} />
        </Box>
        <Box aria-hidden={true} sx={{ flex: 1, minWidth: 0 }}>
          <Box sx={{ fontSize: 'large.fontSize', fontWeight: 600 }}>{name}</Box>
          <Box
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: space.gap,
              fontSize: 'medium.fontSize',
              color: 'text.secondary',
            }}
          >
            <Box
              sx={{
                width: 8,
                height: 8,
                flexShrink: 0,
                borderRadius: '50%',
                bgcolor: DOT_COLOR[tone],
              }}
            />
            <span>{status}</span>
          </Box>
        </Box>
        {action ? (
          <Box sx={{ flexShrink: 0, display: 'flex', gap: space.gap }}>{action}</Box>
        ) : null}
      </Box>
    </SettingRow>
  )
}

/** The rows that belong to one service, set in under its account row. Contents, so each stays its own tile. */
export function ServiceBlock({ children }: { children: ReactNode }) {
  return (
    <Box
      className="settings-contents"
      sx={{ display: 'contents', '& > *:not(:first-of-type)': { ml: 3 } }}
    >
      {children}
    </Box>
  )
}
